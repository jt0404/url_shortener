package main

import (
	"fmt"
	"time"
	"net/http"
	"html/template"
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
        "go.mongodb.org/mongo-driver/mongo/options"
        "go.mongodb.org/mongo-driver/mongo/readpref"
)

var mongoClient *mongo.Client
var mongoErr error
var urlsCollection *mongo.Collection

type Url struct {
	Long 		string
	Short		string
	CreatedAt	time.Time
	ExpiresAt	time.Time
}

type Urls struct {
	Urls []bson.M
}

func encodeLong(long string, id int64) (short string) {
	// base 62 encoding
	s := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	base := int64(len(s))
	for id > 0 {
		short = string(s[id % base]) + short
		id /= base
	}
	return short
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	// if found in db, redirect to long url
	cursor, err := urlsCollection.Find(context.TODO(), bson.M{"short": r.URL.String()[1:]})
	if err != nil {
		panic(err)
	}

	var result []bson.M
	if err = cursor.All(context.TODO(), &result); err != nil {
		panic(err)
	}

	fmt.Println(result)

	for _, res := range result {
		s := fmt.Sprintf("%v", res["long"])
		fmt.Println(s)
		http.Redirect(w, r, s, 301)
		return
	}

	cursor, err = urlsCollection.Find(context.TODO(), bson.D{})
	if err != nil {
		panic(err)
	}

	if err = cursor.All(context.TODO(), &result); err != nil {
		panic(err)
	}

	for _, res := range result {
		s := fmt.Sprintf("%v", res["short"])
		res["short"] = r.Host + "/" + s
	}

	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		panic(err)
	}

	urls := Urls{Urls: result}
	t.Execute(w, urls)
}

func createUrlHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/create_url.html")
	if err != nil {
		panic(err)
	}

	if r.Method != http.MethodPost {
		t.Execute(w, nil)
		return
	}

	// Post request
	count, err := urlsCollection.CountDocuments(context.TODO(), bson.D{})
	if err != nil {
		panic(err)
	}

	long := r.FormValue("url")
	if long == "" {
		return
	}

	cursor, err := urlsCollection.Find(context.TODO(), bson.M{"long": long})
	if err != nil {
		panic(err)
	}

	var list []bson.M
	if err = cursor.All(context.TODO(), &list); err != nil {
		panic(err)
	}
	// if found in db redirect to home page
	if len(list) != 0 {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	short := encodeLong(long, int64(count + 1))
	createdAt := time.Now()
	expiresAt := createdAt.AddDate(0, 1, 0)
	url := Url{Long: long, Short: short, CreatedAt: createdAt, ExpiresAt: expiresAt}

	_, err = urlsCollection.InsertOne(context.TODO(), url)
	if err != nil {
		panic(err)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func main() {
	// setup mongo
	mongoClient, mongoErr = mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017"))
        if mongoErr != nil {
                panic(mongoErr)
        }
	if mongoErr = mongoClient.Ping(context.TODO(), readpref.Primary()); mongoErr != nil {
		panic(mongoErr)
	}
	urlsCollection = mongoClient.Database("urls").Collection("urls")

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/create", createUrlHandler)
	http.ListenAndServe(":8000", nil)
}
