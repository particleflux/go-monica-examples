// Import a CSV-export of "personio" to monica
// note: birth date does not seem to be saved correctly

package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"github.com/goodsign/monday"
	"github.com/particleflux/go-monica/monica"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

func contactExists(client *monica.Client, firstname, lastname string) bool {
	fullName := firstname + " " + lastname
	list, meta, err := client.Contacts.SearchContacts(
		context.Background(),
		&monica.ContactSearchListOptions{
			Query: fullName,
		},
	)

	if err != nil {
		log.Fatal(err)
	}

	if meta.Total == 0 {
		return false
	}

	for _, existing := range *list {
		if existing.FirstName == firstname && existing.LastName == lastname {
			return true
		}
	}

	return false
}

func main() {
	url := flag.String("url", "", "Monica API URL (with /api/)")
	token := flag.String("token", "", "Access token")
	company := flag.String("company", "", "Company to add to contacts")
	genderId := flag.Int("gender", 3, "genderid to set for all contacts")
	tagsStr := flag.String("tags", "", "comma-separated tags to add to contacts")
	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "Usage: %s [options] csv-file\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if len(flag.Args()) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	f, err := os.Open(flag.Args()[0])
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	reader := csv.NewReader(f)
	reader.Comma = ';'
	reader.LazyQuotes = true

	header, err := reader.Read()
	if err != nil {
		log.Fatal(err)
	}

	headerMap := make(map[string]int, len(header))
	for index, name := range header {
		headerMap[name] = index
	}
	fmt.Println(headerMap)
	client := monica.NewClient(*url, *token)

	loc, _ := time.LoadLocation("Europe/Berlin")
	tags := strings.Split(*tagsStr, ",")

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		bd, bdErr := monday.ParseInLocation("02. January", record[headerMap["Geburtstag"]], loc, monday.LocaleDeDE)
		if err != nil {
			log.Println(err)
		}

		contact := monica.ContactInput{
			FirstName:        record[headerMap["Vorname"]],
			LastName:         record[headerMap["Nachname"]],
			IsDeceased:       false,
			IsBirthdateKnown: false,
			GenderId:         *genderId,
		}

		if bdErr == nil {
			contact.IsBirthdateKnown = true
			contact.BirthdateIsAgeBased = false
			contact.BirthdateMonth = int(bd.Month())
			contact.BirthdateDay = bd.Day()
		}

		fmt.Println(contact.FirstName + " " + contact.LastName)
		//fmt.Printf("%#v\n", contact)

		if contactExists(client, contact.FirstName, contact.LastName) {
			fmt.Printf("contact '%s %s' already exists, skipping\n", contact.FirstName, contact.LastName)
			continue
		}

		newContact, err := client.Contacts.CreateContact(context.Background(), &contact)
		if err != nil {
			fmt.Println("error on contact creation: ", err)
			continue
		}

		client.Contacts.UpdateContactCareer(context.Background(), newContact.Id, record[headerMap["Position"]], *company)

		// add email (contactfield type == 1)
		client.Contacts.CreateContactField(context.Background(), &monica.CreateContactFieldInput{
			ContactFieldTypeId: 1,
			ContactId:          newContact.Id,
			Data:               record[headerMap["E-Mail"]],
		})

		if len(tags) > 0 {
			client.Contacts.AddTags(context.Background(), newContact.Id, tags)
		}
	}
}
