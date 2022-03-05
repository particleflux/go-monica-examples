package main

// Go through all contacts in account, interactively asking for gender when it
// is set to "unknown-gender" id

import (
	"context"
	"flag"
	"fmt"
	"github.com/manifoldco/promptui"
	"github.com/particleflux/go-monica/monica"
	"log"
	"os"
)

func main() {
	apiUrl := ""
	flag.StringVar(&apiUrl, "api-url", "", "API base url. Should include '/api/' suffix")
	accessToken := ""
	flag.StringVar(&accessToken, "token", "", "Oauth access token")
	unknownGenderId := 0
	flag.IntVar(&unknownGenderId, "unknown-gender", 3, "contacts with this gender_id will be updated")

	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if apiUrl == "" || accessToken == "" {
		log.Fatal("required API parameters not given")
	}

	client := monica.NewClient(apiUrl, accessToken)

	genders, _, _ := client.Genders.ListGenders(context.Background(), nil)
	genderMap := make(map[int]string, len(*genders))
	genderIdMap := make(map[string]int, len(*genders))
	genderStr := make([]string, len(*genders))
	i := 0
	for _, gender := range *genders {
		genderMap[gender.Id] = gender.Name
		genderIdMap[gender.Name] = gender.Id
		genderStr[i] = gender.Name
		i++
	}

	fmt.Printf("%#v\n", genderMap)

	unknownGender := genderMap[unknownGenderId]

	page := 1
	opts := monica.ContactSearchListOptions{
		ListOptions: monica.ListOptions{
			Page: page,
		},
	}

	prompt := promptui.Select{
		Label: "Select Gender",
		Items: genderStr,
	}

	for {
		contacts, meta, err := client.Contacts.SearchContacts(context.Background(), &opts)
		if err != nil {
			log.Fatal(err)
		}

		for _, contact := range *contacts {
			fmt.Print(contact.FirstName, " ", contact.LastName, " ")

			if contact.Gender == unknownGender || contact.Gender == "" {
				fmt.Println("gender not set")
				_, result, err := prompt.Run()
				if err != nil {
					log.Fatal(err)
				}

				input := monica.ContactToContactInput(*contact)
				input.GenderId = genderIdMap[result]

				_, err = client.Contacts.UpdateContact(context.Background(), contact.Id, input)
				if err != nil {
					fmt.Println(err)
				}
			} else {
				fmt.Println(contact.Gender, " skip")
			}
		}

		if meta.LastPage == meta.CurrentPage {
			break
		}
		opts.Page++
	}
}
