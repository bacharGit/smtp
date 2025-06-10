package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/x/smtp/smtp"
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

	f, err := excelize.OpenFile("split_contacts.xlsx")
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

	// sheets := f.GetSheetList()
	// fmt.Println("Available sheets:", sheets)

	// for si, sheet := range sheets {
	// 	if !strings.HasPrefix(sheet, "sheet_") {
	// 		fmt.Printf("Skipping sheet %s (does not match naming convention)\n", sheet)
	// 		continue
	// 	}

	// 	rows, err := f.GetRows(sheet)
	// 	if err != nil {
	// 		fmt.Printf("Failed to read rows from %s: %v\n", sheet, err)
	// 		continue
	// 	}

	// 	fmt.Printf("🔄 Processing %d rows in sheet %s (%d/%d)\n", len(rows), sheet, si+1, len(sheets))
	// 	sent := 0

	// 	for i, row := range rows {
	// 		if len(row) < 2 {
	// 			fmt.Printf("⚠️ Skipping incomplete row at index %d in %s: %v\n", i, sheet, row)
	// 			continue
	// 		}

	// 		salutation := row[0]
	// 		email := row[1]

	// 		if !strings.HasPrefix(salutation, "Sehr geehrte") {
	// 			salutation = fmt.Sprintf("Sehr geehrte %s", salutation)
	// 		}

	// 		if salutation == "" || email == "" {
	// 			fmt.Printf("⚠️ Skipping row %d in %s due to empty fields\n", i, sheet)
	// 			continue
	// 		}

	// 		personalized := strings.ReplaceAll(templateStr, "{{salutation}}", salutation)

	// 		emailData := map[string]interface{}{
	// 			"html":    personalized,
	// 			"text":    "",
	// 			"subject": "Bewerbung um einen Ausbildungsplatz als Bauzeichner",
	// 			"from":    map[string]string{"name": "Bachar Gmagour", "email": "bewerbung@bachargmagour.com"},
	// 			"to":      []map[string]string{{"email": email}},
	// 		}

	// 		err := client.SMTPSendMail(emailData)
	// 		if err != nil {
	// 			fmt.Printf("❌ Failed to send email to %s: %v\n", email, err)
	// 		} else {
	// 			fmt.Printf("✅ Email sent to %s (sheet: %s, row: %d)\n", email, sheet, i)
	// 			sent++
	// 		}
	// 	}

	// 	fmt.Printf("✅ Finished sheet %s: %d emails sent\n", sheet, sent)

	// 	// Only wait if more sheets are coming
	// 	if si < len(sheets)-1 {
	// 		fmt.Printf("⏳ Waiting 70 minutes before next batch...\n")
	// 		for remaining := cooldown; remaining > 0; remaining -= time.Minute {
	// 			fmt.Printf("🕒 %d minutes remaining...\n", int(remaining.Minutes()))
	// 			time.Sleep(time.Minute)
	// 		}
	// 	}
	// }

	// fmt.Println("🎉 All sheets processed!")

	personalized1 := strings.ReplaceAll(templateStr, "{{salutation}}", "Sehr geehrte Herr Bachar")

	emailData1 := map[string]interface{}{
		"html":    personalized1,
		"text":    "",
		"subject": "Bewerbung um einen Ausbildungsplatz als Bauzeichner",
		"from":    map[string]string{"name": "Bachar Gmagour", "email": "bewerbung@bachargmagour.com"},
		"to":      []map[string]string{{"email": "bacharagmagour2021@gmail.com"}},
	}

	if err := client.SMTPSendMail(emailData1); err != nil {
		fmt.Printf("❌ Failed to send email to %s: %v\n", "bacharagmagour2021@gmail.com", err)
	}

	personalized2 := strings.ReplaceAll(templateStr, "{{salutation}}", "Sehr geehrte Herr Gmagour")

	emailData2 := map[string]interface{}{
		"html":    personalized2,
		"text":    "",
		"subject": "Bewerbung um einen Ausbildungsplatz als Bauzeichner",
		"from":    map[string]string{"name": "Bachar Gmagour", "email": "bewerbung@bachargmagour.com"},
		"to":      []map[string]string{{"email": "support@accountify.ac"}},
	}

	if err := client.SMTPSendMail(emailData2); err != nil {
		fmt.Printf("❌ Failed to send email to %s: %v\n", "support@accountify.ac", err)
	}

	personalized3 := strings.ReplaceAll(templateStr, "{{salutation}}", "Sehr geehrte Herr Brahim")

	emailData3 := map[string]interface{}{
		"html":    personalized3,
		"text":    "",
		"subject": "Bewerbung um einen Ausbildungsplatz als Bauzeichner",
		"from":    map[string]string{"name": "Bachar Gmagour", "email": "bewerbung@bachargmagour.com"},
		"to":      []map[string]string{{"email": "brahime373@gmail.com"}},
	}

	if err := client.SMTPSendMail(emailData3); err != nil {
		fmt.Printf("❌ Failed to send email to %s: %v\n", "brahime373@gmail.com", err)
	}
}
