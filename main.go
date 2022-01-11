package main

import (
	"encoding/csv"
	"errors"
	"io"
	"log"
	"os"
	"regexp"
)

type Action string

const (
	ActionIncome  = "收入"
	ActionExpense = "支出"

	ActionTransactionIn  = "转入"
	ActionTransactionOut = "转出"

	AccountToIn  = "应收款项"
	AccountToOut = "应付款项"
)

// 一笔交易，收入、支出、转账、借贷
type Transaction struct {
	// 交易发生的时间
	Time string

	// 从此账户转出
	From string

	// 转入到此账户
	To string

	// 交易金额
	Value string

	// 描述
	Description string
}

func ProcessSuiShouJi() error {
	sf, err := os.Open("src.csv")
	if err != nil {
		return err
	}
	defer sf.Close()
	r := csv.NewReader(sf)

	wf, err := os.Create("dst.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer wf.Close()
	w := csv.NewWriter(wf)

	for i := 0; i < 2; i++ {
		r.Read()
	}
	r.FieldsPerRecord = 12

	cache := make(map[string]Transaction)

	for {
		// 从随手记导出文件中的每一行载入记录
		// 随手记导出的数据包含 12 个字段
		// 	0: 交易类型
		// 	1: 日期
		// 	2: 类别
		// 	3: 子类别
		// 	4: 项目
		// 	5: 账户
		// 	6: 币种
		// 	7: 金额
		// 	8: 成员
		// 	9: 商家
		// 10: 备注
		// 11: 关联 ID
		record, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return err
		}

		t := Transaction{
			Time:        record[1],
			Value:       record[7],
			Description: record[10],
		}

		switch record[0] {
		case ActionIncome:
			t.From, t.To = record[2]+":"+record[3], record[5]
		case ActionExpense:
			t.From, t.To = record[5], record[2]+":"+record[3]
		case ActionTransactionIn:
			AssociationId := record[11]
			t.To = record[5]
			if tt := cache[AssociationId]; tt.From == "" {
				cache[AssociationId] = t
				continue
			} else {
				t.From = tt.From
			}
		case ActionTransactionOut:
			AssociationId := record[11]
			t.From = record[5]
			if tt := cache[AssociationId]; tt.To == "" {
				cache[AssociationId] = t
				continue
			} else {
				t.To = tt.To
			}
		default:
			// log.Printf("undefined transaction type: %s", record[0])
			continue
		}

		// 处理债务
		if t.From == AccountToIn || t.From == AccountToOut || t.To == AccountToIn || t.To == AccountToOut {
			if account, description := ParseLoanDescription(t.Description); account != "" {
				t.Description = description
				if t.From == AccountToIn || t.From == AccountToOut {
					t.From = account
				}
				if t.To == AccountToIn || t.To == AccountToOut {
					t.To = account
				}
			}
		}

		if err := w.Write([]string{
			t.Time,
			t.From,
			t.To,
			t.Value,
			t.Description,
		}); err != nil {
			return err
		}
	}
	w.Flush()
	return nil
}

func ParseLoanDescription(desc string) (string, string) {
	re := regexp.MustCompile(`^#(借入|借出|还债|收债|免债|坏账): (.*)# ?(.*)`)
	if ss := re.FindStringSubmatch(desc); len(ss) == 4 {
		return ss[2], ss[3]
	}

	aa := regexp.MustCompile(`^\[(借入|借出|还债|收债|免债|坏账)\](.*) ?(.*)`)
	if ss := aa.FindStringSubmatch(desc); len(ss) == 4 {
		return ss[2], ss[3]
	}
	return "", ""
}

func main() {
	if err := ProcessSuiShouJi(); err != nil {
		log.Fatalf("process loan fail: %s", err)
	}
}
