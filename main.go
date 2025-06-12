package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/x/smtp/smtp"

	"github.com/joho/godotenv"
	"github.com/xuri/excelize/v2"
)

const cooldown = 70 * time.Minute

func main() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}

	template, err := os.ReadFile("template.html")
	if err != nil {
		panic(fmt.Errorf("failed to read template: %w", err))
	}
	templateStr := string(template)

	f, err := excelize.OpenFile("out.xlsx")
	if err != nil {
		panic(fmt.Errorf("failed to open Excel file: %w", err))
	}
	defer f.Close()

	clientId := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")

	client := smtp.NewClient(clientId, clientSecret, "tokens")
	if err := client.Init(); err != nil {
		panic(err)
	}

	sheets := f.GetSheetList()

	for si, sheet := range sheets {
		fmt.Printf("📄 Processing sheet: %s\n", sheet)
		rows, err := f.GetRows(sheet)
		if err != nil {
			fmt.Printf("❌ Failed to read rows from sheet %s: %v\n", sheet, err)
			continue
		}

		if len(rows) < 2 {
			fmt.Printf("⚠️  Sheet %s has no emails to send.\n", sheet)
			continue
		}

		sent := 0
		for i := 1; i < len(rows) && i <= 50; i++ { // Skip first row (x), send next 50
			if len(rows[i]) == 0 {
				continue
			}
			email := strings.TrimSpace(rows[i][0])
			if email == "" {
				continue
			}

			emailData := map[string]interface{}{
				"html":    templateStr,
				"text":    "",
				"subject": "Bewerbung um einen Ausbildungsplatz als Bauzeichner",
				"from":    map[string]string{"name": "Bachar Gmagour", "email": "bewerbung@bachargmagour.com"},
				"to":      []map[string]string{{"email": email}},
			}

			err := client.SMTPSendMail(emailData)
			if err != nil {
				fmt.Printf("❌ Failed to send email to %s: %v\n", email, err)
			} else {
				fmt.Printf("✅ Email sent to %s (sheet: %s, row: %d)\n", email, sheet, i+1)
				sent++
			}
		}

		fmt.Printf("✅ Finished sheet %s: %d emails sent\n", sheet, sent)

		// Wait before next batch
		if si < len(sheets)-1 {
			fmt.Printf("⏳ Waiting 70 minutes before next batch...\n")
			for remaining := cooldown; remaining > 0; remaining -= time.Minute {
				fmt.Printf("🕒 %d minutes remaining...\n", int(remaining.Minutes()))
				time.Sleep(time.Minute)
			}
		}
	}

	fmt.Println("🎉 All sheets processed!")
}
