package ai

import "fmt"

func (res *AIResult) Validate() error {
	if res.Category == "" {
		return fmt.Errorf("category is required")
	}

	if res.Summary ==  "" || len(res.Summary) < 10 {
		return fmt.Errorf("summary is too short or missing")
	}

	if res.Category == "Internship" || res.Category == "Full-time" {
		if res.Company == nil || *res.Company == "" {
			return fmt.Errorf("company name missing for job category : %v", res.Category)
		}
	}
	return nil
}
