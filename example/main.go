package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/hsedr/pocketbase"
	"github.com/mitchellh/mapstructure"
)

type Post struct {
	Field string `json:"field"`
	ID    string `json:"id"`
}

type Model struct {
	ID             string `json:"id"`
	CollectionID   string `json:"collectionId"`
	CollectionName string `json:"collectionName"`
	Created        string `json:"created"`
	Updated        string `json:"updated"`
	Username       string `json:"username"`
	Email          string `json:"email"`
	Verified       bool   `json:"verified"`
	EmailVisibilty bool   `json:"emailVisibility"`
}

func main() {
	// REMEMBER to start the Pocketbase before running this example with `make serve` command

	var errs error
	client := pocketbase.NewClient("http://localhost:8090",
		pocketbase.WithUserEmailPassword("user@user.com", "user@user.com"),
	)
	// Other configuration options:
	// pocketbase.WithAdminEmailPassword("admin@admin.com", "admin@admin.com")
	// pocketbase.WithUserEmailPassword("user@user.com", "user@user.com")
	// pocketbase.WithUserToken(token)
	// pocketbase.WithAdminToken(token)
	// pocketbase.WithDebug()

	err := client.Authorize()
	if err != nil {
		log.Fatal(err)
	}

	model := Model{}
	err = client.AuthStore().Model(&model)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Model:", model)

	response, err := client.List("posts_public", pocketbase.ParamsList{
		Size:    1,
		Page:    1,
		Sort:    "-created",
		Filters: "field~'test'",
	})

	errs = errors.Join(errs, err)

	log.Printf("Total items: %d, total pages: %d\n", response.TotalItems, response.TotalPages)
	for _, item := range response.Items {
		var test Post
		err := mapstructure.Decode(item, &test)
		errs = errors.Join(errs, err)

		log.Printf("Item: %#v\n", test)
	}

	log.Println("Inserting new item")
	// you can use struct type - just make sure it has JSON tags
	_, err = client.Create("posts_public", Post{
		Field: "test_" + time.Now().Format(time.Stamp),
	})
	errs = errors.Join(errs, err)

	// or you can use simple map[string]any
	r, err := client.Create("posts_public", map[string]any{
		"field": "test_" + time.Now().Format(time.Stamp),
	})
	errs = errors.Join(errs, err)

	err = client.Delete("posts_public", r.ID)
	errs = errors.Join(errs, err)

	if errs != nil {
		log.Fatal(errs)
	}
}
