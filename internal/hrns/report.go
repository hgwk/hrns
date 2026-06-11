package hrns

import (
	"fmt"
	"os"
)

type Finding struct {
	Message string
	Detail  string
}

type Audit struct {
	Name     string
	Findings []Finding
}

func NewAudit(name string) *Audit {
	return &Audit{Name: name}
}

func (a *Audit) Fail(message string, detail ...string) {
	d := ""
	if len(detail) > 0 {
		d = detail[0]
	}
	a.Findings = append(a.Findings, Finding{Message: message, Detail: d})
}

func (a *Audit) Finish() error {
	if len(a.Findings) == 0 {
		fmt.Printf("%s: PASS\n", a.Name)
		return nil
	}
	fmt.Fprintf(os.Stderr, "%s: FAIL (%d finding(s))\n", a.Name, len(a.Findings))
	for _, finding := range a.Findings {
		fmt.Fprintln(os.Stderr, "- "+finding.Message)
		if finding.Detail != "" {
			fmt.Fprintln(os.Stderr, "  "+finding.Detail)
		}
	}
	return fmt.Errorf("%s failed", a.Name)
}

func finishByMode(a *Audit, findings []Finding, mode string) error {
	if mode == "" {
		mode = "fail"
	}
	if len(findings) == 0 {
		return a.Finish()
	}
	if mode == "off" {
		fmt.Println("audit disabled by config")
		return nil
	}
	if mode == "warn" {
		for _, finding := range findings {
			fmt.Fprintf(os.Stderr, "%s: WARN - %s\n", a.Name, finding.Message)
			if finding.Detail != "" {
				fmt.Fprintln(os.Stderr, "  "+finding.Detail)
			}
		}
		return a.Finish()
	}
	for _, finding := range findings {
		a.Fail(finding.Message, finding.Detail)
	}
	return a.Finish()
}
