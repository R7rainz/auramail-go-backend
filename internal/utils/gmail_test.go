package utils

import "testing"

func TestCleanTextForAi(t *testing.T){
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {
            name:     "Removes extra whitespace",
            input:    "Hello    World  \n  Test",
            expected: "Hello World Test",
        },
        {
            name:     "Truncates long text",
            input:    string(make([]byte, 3000)), // 3000 characters
            expected: "... [truncated]",           // Should end with this
        },
    }

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanTextForAi(tt.input)

		    //checking logic
			if tt.name == "Removes extra whitespace" && result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}

			if tt.name == "Truncates long text" && len(result) > 2015 {
				t.Error("Text was not truncated correctly")
			} 
		})
	}
}
