package main

import layaonnx "github.com/MstyAI/laya-onnx"

func benchmarkQuestions() (map[string]layaonnx.Question, error) {
	department, err := layaonnx.NewChoice(
		"Which department should handle this customer request?",
		layaonnx.Option{Name: "account", Description: "Sign-in, identity, access, and account settings"},
		layaonnx.Option{Name: "billing", Description: "Charges, invoices, payments, and refunds"},
		layaonnx.Option{Name: "sales", Description: "Pricing, plans, purchasing, and product evaluation"},
		layaonnx.Option{Name: "shipping", Description: "Delivery, tracking, damaged packages, and returns in transit"},
		layaonnx.Option{Name: "technical", Description: "Bugs, outages, errors, and product malfunctions"},
	)
	if err != nil {
		return nil, err
	}
	refund, err := layaonnx.NewNoul("Does the customer explicitly request a refund or their money back?")
	if err != nil {
		return nil, err
	}
	urgency, err := layaonnx.NewScore(
		"How urgent is the request?",
		"routine; no time pressure",
		"soon; should be handled within a few days",
		"urgent; blocking work or needs action today",
		"critical; severe security, safety, or widespread outage",
	)
	if err != nil {
		return nil, err
	}
	return map[string]layaonnx.Question{
		"department": department,
		"refund":     refund,
		"urgency":    urgency,
	}, nil
}
