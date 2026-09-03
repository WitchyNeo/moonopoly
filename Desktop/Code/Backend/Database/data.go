package MBSEGames

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"

	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//Intialisation
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func InitializeFirebaseApp(ctx context.Context, serviceAccountKeyPath string) (*firebase.App, *auth.Client, error) {
	opt := option.WithCredentialsFile(serviceAccountKeyPath)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, nil, err
	}

	client, err := app.Auth(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting Firebase Auth client: %w", err) // Return error for main
	}

	return app, client, nil
}

func GetFirestoreClient(ctx context.Context, app *firebase.App) (*firestore.Client, error) {
	FirestoreClient, err := app.Firestore(ctx)
	if err != nil {
		return nil, err
	}
	return FirestoreClient, nil
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//Structures
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
type User struct {
	DocRef       string
	UID          string
	FirstName    string
	Surname      string
	Email        string
	CelNum       string
	DisplayName  string
	PicID        string
	Streak       int
	TotalScore   int64
	TotalTime    int64
	RecordStreak int
	IsAdmin      bool
	LastPlayed   time.Time
}

type Question struct {
	DocRef      string
	Subject     string
	Category    string
	Outcome     int
	Level       int
	Question    string
	Option1     string
	Option2     string
	Option3     string
	Option4     string
	Answer      int
	Explanation string
}

type Session struct {
	DocRef           string
	UserDisplayName  string
	Subject          string
	Gamemode         string
	Ai1              string
	Ai2              string
	Ai3              string
	StartTime        string
	EndTime          string
	Score            int64
	NumCorrect   int
	NumIncorrect int
	QDetails         string
}

type ChanceCard struct {
	DocRef        string
	Message       string
	IsPosChange   bool
	IsScoreChange bool
	IsToFieldMove bool
	Value         int
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//Global Variables & Constants
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
var Adjectives = [...]string{"able", "afraid", "angry", "awake", "bad", "beautiful", "big", "bitter", "black", "blue", "bold", "bored", "brave", "bright", "broad", "broken", "brown", "busy", "calm", "certain", "cheap", "clean", "clear", "clever", "close", "cloudy", "cold", "cool", "crazy", "cruel", "dark", "dead", "deep", "difficult", "dirty", "dry", "dull", "early", "easy", "empty", "equal", "excited", "faint", "false", "famous", "far", "fast", "fat", "few", "fine", "firm", "first", "flat", "foreign", "free", "fresh", "full", "funny", "gentle", "glad", "gold", "good", "gray", "great", "green", "happy", "hard", "heavy", "high", "hollow", "hot", "huge", "hungry", "ill", "important", "kind", "large", "last", "late", "lazy", "left", "light", "little", "lively", "lonely", "long", "loud", "low", "mad", "main", "many", "narrow", "near", "neat", "necessary", "new", "next", "nice", "noisy", "normal", "old", "open", "orange", "other", "pale", "past", "pink", "plain", "pleasant", "poor", "polite", "possible", "present", "pretty", "proud", "public", "purple", "quick", "quiet", "rare", "ready", "real", "red", "rich", "right", "rough", "round", "sad", "safe", "salty", "same", "scared", "secret", "serious", "sharp", "short", "shy", "sick", "silent", "silly", "simple", "single", "slow", "small", "smooth", "soft", "solid", "sour", "special", "square", "steady", "steep", "stiff", "strange", "strict", "strong", "stupid", "sudden", "sure", "sweet", "tall", "tame", "tender", "thick", "thin", "thirsty", "tidy", "tight", "tiny", "tired", "true", "ugly", "useful", "usual", "vague", "violet", "warm", "weak", "wet", "white", "whole", "wicked", "wide", "wild", "wise", "wonderful", "wrong", "yellow", "young"}
var Nouns = [...]string{"air", "animal", "answer", "apple", "area", "arm", "art", "baby", "back", "bag", "ball", "bank", "bed", "bird", "blood", "boat", "body", "bone", "book", "bottom", "box", "boy", "branch", "bread", "brother", "building", "bus", "car", "card", "case", "cat", "cause", "chair", "child", "city", "class", "cloud", "coat", "color", "company", "corner", "country", "course", "cow", "crowd", "cup", "day", "desk", "dog", "door", "dream", "dress", "ear", "earth", "egg", "end", "eye", "face", "fact", "family", "father", "fear", "field", "figure", "finger", "fire", "fish", "floor", "flower", "food", "foot", "force", "forest", "friend", "front", "fruit", "game", "garden", "girl", "glass", "gold", "government", "grass", "ground", "group", "hair", "hand", "hat", "head", "health", "heart", "hill", "home", "horse", "hospital", "hour", "house", "ice", "idea", "image", "industry", "island", "job", "key", "kind", "king", "kitchen", "lake", "land", "language", "law", "leaf", "leg", "letter", "level", "life", "light", "line", "list", "love", "machine", "man", "map", "market", "material", "matter", "meal", "meat", "member", "message", "metal", "milk", "mind", "minute", "money", "month", "moon", "morning", "mother", "mountain", "mouth", "music", "name", "nation", "neck", "needle", "neighbor", "network", "news", "night", "noise", "north", "nose", "note", "number", "ocean", "office", "oil", "order", "page", "paint", "paper", "parent", "park", "part", "party", "past", "path", "pen", "pencil", "people", "person", "phone", "picture", "piece", "pig", "place", "plan", "plane", "plant", "plate", "point", "police", "pool", "power", "president", "price", "problem", "product", "program", "property", "question", "radio", "rain", "reason", "record", "report", "rest", "result", "rice", "river", "road", "rock", "room", "root", "rule", "salt", "sand", "school", "science", "sea", "seat", "second", "seed", "sentence", "service", "shape", "sheep", "ship", "shirt", "shoe", "shop", "show", "side", "sign", "silver", "sister", "size", "skin", "sky", "snow", "soap", "society", "soil", "son", "song", "sound", "soup", "south", "space", "spoon", "sport", "spring", "square", "star", "state", "station", "steam", "steel", "step", "stick", "stone", "stop", "store", "story", "street", "student", "study", "sugar", "summer", "sun", "system", "table", "tail", "tea", "teacher", "team", "television", "test", "thing", "thought", "time", "toe", "top", "town", "toy", "track", "trade", "train", "tree", "trip", "trouble", "truck", "turn", "tv", "uncle", "unit", "valley", "value", "video", "view", "village", "voice", "wall", "war", "water", "way", "week", "weight", "west", "wheel", "wind", "window", "wine", "winter", "wire", "woman", "wood", "word", "work", "world", "yard", "year"}

type contextKey string

const userContextKey contextKey = "firebaseUser"

var firebaseAuth *auth.Client
var CurrentUser User

var Users []User
var Questions []Question
var ChanceCards []ChanceCard
var Sessions []Session

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//Users
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//Auth middleware
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//Ensure firebaseAuth client is initialized before proceeding
		if firebaseAuth == nil {
			http.Error(w, "Internal Server Error - Auth not ready", http.StatusInternalServerError)
			return
		}

		//Get the ID Token from the Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized: Missing Authorization Header", http.StatusUnauthorized)
			return
		}

		//Expecting "Bearer <token>"
		splitToken := strings.Split(authHeader, "Bearer ")
		if len(splitToken) != 2 || splitToken[1] == "" {
			http.Error(w, "Unauthorized: Invalid Authorization Header format", http.StatusUnauthorized)
			return
		}
		idToken := splitToken[1]

		//Verify the ID token using the Firebase Admin SDK
		token, err := firebaseAuth.VerifyIDToken(r.Context(), idToken)
		if err != nil {
			// Provide a generic error message to the client
			http.Error(w, "Unauthorized: Invalid or expired token", http.StatusUnauthorized)
			return
		}

		ctxWithUser := context.WithValue(r.Context(), userContextKey, token)
		r = r.WithContext(ctxWithUser) //Create a new request with the updated context

		//Proceed to the next handler in the chain
		next.ServeHTTP(w, r)
	})
}

func protectedHandler(w http.ResponseWriter, r *http.Request) {
	//Retrieve the verified token (user info) from the context.
	//This was set by the AuthMiddleware.
	token, ok := r.Context().Value(userContextKey).(*auth.Token)
	if !ok || token == nil {
		http.Error(w, "Internal Server Error - User context missing", http.StatusInternalServerError)
		return
	}

	//--- Access User Information ---
	CurrentUser.UID = token.UID

	//Respond to the client
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

//Login
func Login(ctx context.Context, app *firebase.App, firestoreClient *firestore.Client, uid string) error {
	docGrab, err := firestoreClient.Collection("users").Doc(uid).Get(ctx)
	if err != nil {
		return err
	}

	var u User
	docGrab.DataTo(&u)

	CurrentUser.DocRef = uid
	CurrentUser.UID = uid
	CurrentUser.FirstName = u.FirstName
	CurrentUser.Surname = u.Surname
	CurrentUser.Email = u.Email
	CurrentUser.CelNum = u.CelNum
	CurrentUser.DisplayName = u.DisplayName
	CurrentUser.PicID = u.PicID
	CurrentUser.Streak = u.Streak
	CurrentUser.TotalScore = u.TotalScore
	CurrentUser.TotalTime = u.TotalTime
	CurrentUser.RecordStreak = u.RecordStreak
	CurrentUser.IsAdmin = u.IsAdmin
	CurrentUser.LastPlayed = u.LastPlayed

	err = GetQuestions(ctx, firestoreClient)
	if err != nil {
		return err
	}

	err = GetChanceCards(ctx, firestoreClient)
	if err != nil {
		return err
	}

	if CurrentUser.IsAdmin {
		err = GetUsers(ctx, firestoreClient)
		if err != nil {
			return err
		}

		err = GetSessions(ctx, firestoreClient)
		if err != nil {
			return err
		}
	}

	return nil

}

func SignUp(ctx context.Context, app *firebase.App, firestoreClient *firestore.Client, email, password, firstname, surname, celnum, displayname string, isadmin bool) error {
	const picid string = "default"

	if displayname == "" {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		displayname = Adjectives[r.Intn(len(Adjectives))] + Nouns[r.Intn(len(Nouns))] + strconv.Itoa(r.Intn(10)) + strconv.Itoa(r.Intn(10)) + strconv.Itoa(r.Intn(10)) + strconv.Itoa(r.Intn(10))
	}

	client, err := app.Auth(ctx)
	if err != nil {
		return err
	}

	params := (&auth.UserToCreate{}).
		Email(email).
		Password(password).
		DisplayName(displayname)

	user, err := client.CreateUser(ctx, params)
	if err != nil {
		return err
	}

	data := map[string]interface{}{
		"uid":          user.UID,
		"firstname":    firstname,
		"surname":      surname,
		"email":        email,
		"celnum":       celnum,
		"displayname":  displayname,
		"picid":        picid, //get a generic pic to place here
		"streak":       0,
		"totalscore":   0,
		"totaltime":    0,
		"recordstreak": 0,
		"isadmin":      isadmin,
		"lastplayed":   time.Now(),
	}
	userRef, _, err := firestoreClient.Collection("users").Add(ctx, data)
	if err != nil {
		return err
	}

	var u User
	u.DocRef = userRef.ID
	u.UID = user.UID
	u.FirstName = firstname
	u.Surname = surname
	u.Email = email
	u.CelNum = celnum
	u.DisplayName = displayname
	u.PicID = picid
	u.Streak = 0
	u.TotalScore = 0
	u.TotalTime = 0
	u.RecordStreak = 0
	u.IsAdmin = isadmin
	u.LastPlayed = time.Now()

	Users = append(Users, u)

	return nil

}

//List
func GetUsers(ctx context.Context, firestoreClient *firestore.Client) error {
	i := 0
	iter := firestoreClient.Collection("users").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err != nil {
			if err.Error() == "no more items in iterator" {
				break
			}
			return err
		}

		var user, u User
		doc.DataTo(&u)
		user.DocRef = doc.Ref.ID
		user.UID = u.UID
		user.FirstName = u.FirstName
		user.Surname = u.Surname
		user.Email = u.Email
		user.CelNum = u.CelNum
		user.DisplayName = u.DisplayName
		user.PicID = u.PicID
		user.Streak = u.Streak
		user.TotalScore = u.TotalScore
		user.TotalTime = u.TotalTime
		user.RecordStreak = u.RecordStreak
		user.IsAdmin = u.IsAdmin
		user.LastPlayed = u.LastPlayed

		if len(Users) > i {
			Users[i] = user
		} else {
			Users = append(Users, user)
		}

		i++
	}

	return nil
}

//Edit User
func EditUser(ctx context.Context, client *firestore.Client, DocRef, Field, NewstrValue string, NewintValue int, Newint64Value int64) (error, string) {
	for i := range Users {
		if Users[i].DocRef == DocRef {
			switch Field {

			case "FirstName":
				Users[i].FirstName = NewstrValue
				//Create an array of updates
				updates := []firestore.Update{
					{Path: "firstname", Value: NewstrValue},
				}

				//Apply the updates
				_, err := client.Collection("users").Doc(DocRef).Update(ctx, updates)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						return fmt.Errorf("user %s not found", DocRef), ""
					}
					return fmt.Errorf("error updating user: %v", err), ""
				}

			case "Surname":
				Users[i].Surname = NewstrValue
				//Create an array of updates
				updates := []firestore.Update{
					{Path: "surname", Value: NewstrValue},
				}

				//Apply the updates
				_, err := client.Collection("users").Doc(DocRef).Update(ctx, updates)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						return fmt.Errorf("user %s not found", DocRef), ""
					}
					return fmt.Errorf("error updating user: %v", err), ""
				}

			case "CelNum":
				Users[i].CelNum = NewstrValue
				//Create an array of updates
				updates := []firestore.Update{
					{Path: "celnum", Value: NewstrValue},
				}

				//Apply the updates
				_, err := client.Collection("users").Doc(DocRef).Update(ctx, updates)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						return fmt.Errorf("user %s not found", DocRef), ""
					}
					return fmt.Errorf("error updating user: %v", err), ""
				}

			case "DisplayName":
				if NewstrValue == "generate" {
					r := rand.New(rand.NewSource(time.Now().UnixNano()))
					displayname := Adjectives[r.Intn(len(Adjectives))] + Nouns[r.Intn(len(Nouns))] + strconv.Itoa(r.Intn(10)) + strconv.Itoa(r.Intn(10)) + strconv.Itoa(r.Intn(10)) + strconv.Itoa(r.Intn(10))
					Users[i].DisplayName = displayname
					//Create an array of updates
					updates := []firestore.Update{
						{Path: "displayname", Value: displayname},
					}

					//Apply the updates
					_, err := client.Collection("users").Doc(DocRef).Update(ctx, updates)
					if err != nil {
						if status.Code(err) == codes.NotFound {
							return fmt.Errorf("user %s not found", DocRef), ""
						}
						return fmt.Errorf("error updating user: %v", err), ""
					}

					return nil, displayname
				} else {
					Users[i].DisplayName = NewstrValue
					//Create an array of updates
					updates := []firestore.Update{
						{Path: "displayname", Value: NewstrValue},
					}

					//Apply the updates
					_, err := client.Collection("users").Doc(DocRef).Update(ctx, updates)
					if err != nil {
						if status.Code(err) == codes.NotFound {
							return fmt.Errorf("user %s not found", DocRef), ""
						}
						return fmt.Errorf("error updating user: %v", err), ""
					}
				}

			case "PicID":
				Users[i].PicID = NewstrValue
				//Create an array of updates
				updates := []firestore.Update{
					{Path: "picid", Value: NewstrValue},
				}

				//Apply the updates
				_, err := client.Collection("users").Doc(DocRef).Update(ctx, updates)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						return fmt.Errorf("user %s not found", DocRef), ""
					}
					return fmt.Errorf("error updating user: %v", err), ""
				}

			case "Streak":
				Users[i].Streak = NewintValue
				//Create an array of updates
				updates := []firestore.Update{
					{Path: "streak", Value: NewintValue},
				}

				//Apply the updates
				_, err := client.Collection("users").Doc(DocRef).Update(ctx, updates)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						return fmt.Errorf("user %s not found", DocRef), ""
					}
					return fmt.Errorf("error updating user: %v", err), ""
				}

			case "TotalScore":
				Users[i].TotalScore = Newint64Value
				//Create an array of updates
				updates := []firestore.Update{
					{Path: "totalscore", Value: Newint64Value},
				}

				//Apply the updates
				_, err := client.Collection("users").Doc(DocRef).Update(ctx, updates)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						return fmt.Errorf("user %s not found", DocRef), ""
					}
					return fmt.Errorf("error updating user: %v", err), ""
				}

			case "TotalTime":
				Users[i].TotalTime = Newint64Value
				//Create an array of updates
				updates := []firestore.Update{
					{Path: "totaltime", Value: Newint64Value},
				}

				//Apply the updates
				_, err := client.Collection("users").Doc(DocRef).Update(ctx, updates)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						return fmt.Errorf("user %s not found", DocRef), ""
					}
					return fmt.Errorf("error updating user: %v", err), ""
				}

			case "RecordStreak":
				Users[i].RecordStreak = NewintValue
				//Create an array of updates
				updates := []firestore.Update{
					{Path: "recordstreak", Value: NewintValue},
				}

				//Apply the updates
				_, err := client.Collection("users").Doc(DocRef).Update(ctx, updates)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						return fmt.Errorf("user %s not found", DocRef), ""
					}
					return fmt.Errorf("error updating user: %v", err), ""
				}

			case "LastPlayed":
				tnow := time.Now()
				Users[i].LastPlayed = tnow

				//Create an array of updates
				updates := []firestore.Update{
					{Path: "lastplayed", Value: tnow},
				}

				//Apply the updates
				_, err := client.Collection("users").Doc(DocRef).Update(ctx, updates)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						return fmt.Errorf("user %s not found", DocRef), ""
					}
					return fmt.Errorf("error updating user: %v", err), ""
				}
			}

			break
		}
	}
	return nil, ""
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//Questions
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//Create
func AddQuestion(ctx context.Context, firestoreClient *firestore.Client, Subject, Category, QuestionText, Option1, Option2, Option3, Option4, Explanation string, Outcome, Level, Answer int) error {
	data := map[string]interface{}{
		"subject":     Subject,
		"category":    Category,
		"outcome":     Outcome,
		"level":       Level,
		"question":    QuestionText,
		"option1":     Option1,
		"option2":     Option2,
		"option3":     Option3,
		"option4":     Option4,
		"answer":      Answer,
		"explanation": Explanation,
	}

	docRef, _, err := firestoreClient.Collection("questions").Add(ctx, data)
	if err != nil {
		return err
	}

	var question Question
	question.DocRef = docRef.ID
	question.Subject = Subject
	question.Category = Category
	question.Outcome = Outcome
	question.Level = Level
	question.Question = QuestionText
	question.Option1 = Option1
	question.Option2 = Option2
	question.Option3 = Option3
	question.Option4 = Option4
	question.Answer = Answer
	question.Explanation = Explanation

	Questions = append(Questions, question)

	return nil
}

//List
func GetQuestions(ctx context.Context, firestoreClient *firestore.Client) error {
	i := 0
	iter := firestoreClient.Collection("questions").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err != nil {
			if err.Error() == "no more items in iterator" {
				break
			}
			return err
		}

		qRef := doc.Ref.ID
		var q Question
		doc.DataTo(&q)
		q.DocRef = qRef

		if len(Questions) > i {
			Questions[i] = q
		} else {
			Questions = append(Questions, q)
		}

		i++

	}

	return nil
}

//Edit Question
func EditQuestion(ctx context.Context, client *firestore.Client, DocRef, Field, NewstrValue string, NewintValue, NewAnswer int) error {
	for i := range Questions {
		if Questions[i].DocRef == DocRef {
			switch Field {
			case "Subject":
				Questions[i].Subject = NewstrValue

				//Create an array of updates
				updates := []firestore.Update{
					{Path: "subject", Value: NewstrValue},
				}

				//Apply the updates
				_, err := client.Collection("questions").Doc(DocRef).Update(ctx, updates)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						return fmt.Errorf("question %s not found", DocRef)
					}
					return fmt.Errorf("error updating question: %v", err)
				}

			case "Category":
				Questions[i].Category = NewstrValue

				//Create an array of updates
				updates := []firestore.Update{
					{Path: "category", Value: NewstrValue},
				}

				//Apply the updates
				_, err := client.Collection("questions").Doc(DocRef).Update(ctx, updates)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						return fmt.Errorf("question %s not found", DocRef)
					}
					return fmt.Errorf("error updating question: %v", err)
				}

			case "Outcome":
				Questions[i].Outcome = NewintValue

				//Create an array of updates
				updates := []firestore.Update{
					{Path: "outcome", Value: NewintValue},
				}

				//Apply the updates
				_, err := client.Collection("questions").Doc(DocRef).Update(ctx, updates)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						return fmt.Errorf("question %s not found", DocRef)
					}
					return fmt.Errorf("error updating question: %v", err)
				}

			case "Level":
				Questions[i].Level = NewintValue

				//Create an array of updates
				updates := []firestore.Update{
					{Path: "level", Value: NewintValue},
				}

				//Apply the updates
				_, err := client.Collection("questions").Doc(DocRef).Update(ctx, updates)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						return fmt.Errorf("question %s not found", DocRef)
					}
					return fmt.Errorf("error updating question: %v", err)
				}

			case "QuestionText":
				Questions[i].Question = NewstrValue

				//Create an array of updates
				updates := []firestore.Update{
					{Path: "question", Value: NewstrValue},
				}

				//Apply the updates
				_, err := client.Collection("questions").Doc(DocRef).Update(ctx, updates)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						return fmt.Errorf("question %s not found", DocRef)
					}
					return fmt.Errorf("error updating question: %v", err)
				}

			case "Option1":
				Questions[i].Option1 = NewstrValue

				//Create an array of updates
				updates := []firestore.Update{
					{Path: "option1", Value: NewstrValue},
				}

				//Apply the updates
				_, err := client.Collection("questions").Doc(DocRef).Update(ctx, updates)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						return fmt.Errorf("question %s not found", DocRef)
					}
					return fmt.Errorf("error updating question: %v", err)
				}

			case "Option2":
				Questions[i].Option2 = NewstrValue

				//Create an array of updates
				updates := []firestore.Update{
					{Path: "option2", Value: NewstrValue},
				}

				//Apply the updates
				_, err := client.Collection("questions").Doc(DocRef).Update(ctx, updates)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						return fmt.Errorf("question %s not found", DocRef)
					}
					return fmt.Errorf("error updating question: %v", err)
				}

			case "Option3":
				Questions[i].Option3 = NewstrValue

				//Create an array of updates
				updates := []firestore.Update{
					{Path: "option3", Value: NewstrValue},
				}

				//Apply the updates
				_, err := client.Collection("questions").Doc(DocRef).Update(ctx, updates)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						return fmt.Errorf("question %s not found", DocRef)
					}
					return fmt.Errorf("error updating question: %v", err)
				}

			case "Option4":
				Questions[i].Option4 = NewstrValue

				//Create an array of updates
				updates := []firestore.Update{
					{Path: "option4", Value: NewstrValue},
				}

				//Apply the updates
				_, err := client.Collection("questions").Doc(DocRef).Update(ctx, updates)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						return fmt.Errorf("question %s not found", DocRef)
					}
					return fmt.Errorf("error updating question: %v", err)
				}

			case "Answer":
				Questions[i].Answer = NewAnswer

				//Create an array of updates
				updates := []firestore.Update{
					{Path: "answer", Value: NewAnswer},
				}

				//Apply the updates
				_, err := client.Collection("questions").Doc(DocRef).Update(ctx, updates)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						return fmt.Errorf("question %s not found", DocRef)
					}
					return fmt.Errorf("error updating question: %v", err)
				}

			case "Explanation":
				Questions[i].Explanation = NewstrValue

				//Create an array of updates
				updates := []firestore.Update{
					{Path: "explanation", Value: NewstrValue},
				}

				//Apply the updates
				_, err := client.Collection("questions").Doc(DocRef).Update(ctx, updates)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						return fmt.Errorf("question %s not found", DocRef)
					}
					return fmt.Errorf("error updating question: %v", err)
				}

			}

			break
		}
	}
	return nil
}

//Delete
func DelQuestion(ctx context.Context, client *firestore.Client, DocRef string) error {
	_, err := client.Collection("questions").Doc(DocRef).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("question %s not found", DocRef)
		}
		return fmt.Errorf("error deleting question: %v", err)
	}

	i := 0
	var tempQuestions []Question
	for {
		if i == len(Questions) {
			return fmt.Errorf("question %s not found", DocRef)
		}

		if Questions[i].DocRef == DocRef {
			tempQuestions = append(tempQuestions, Questions[0:i]...)
			tempQuestions = append(tempQuestions, Questions[i+1:]...)
			break
		}

		i++
	}

	Questions = tempQuestions

	return nil
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//Chance Cards
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//Create
func AddChanceCard(ctx context.Context, firestoreClient *firestore.Client, Message string, Value int, IsPosChange, IsScoreChange, IsToFieldMove bool) error {
	data := map[string]interface{}{
		"message":       Message,
		"isposchange":   IsPosChange,
		"isscorechange": IsScoreChange,
		"istofieldmove": IsToFieldMove,
		"value":         Value,
	}

	docRef, _, err := firestoreClient.Collection("tac").Add(ctx, data)
	if err != nil {
		return err
	}

	var chancecard ChanceCard
	chancecard.DocRef = docRef.ID
	chancecard.Message = Message
	chancecard.IsPosChange = IsPosChange
	chancecard.IsScoreChange = IsScoreChange
	chancecard.IsToFieldMove = IsToFieldMove
	chancecard.Value = Value

	ChanceCards = append(ChanceCards, chancecard)

	return nil
}

//List
func GetChanceCards(ctx context.Context, firestoreClient *firestore.Client) error {
	i := 0
	iter := firestoreClient.Collection("tac").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err != nil {
			if err.Error() == "no more items in iterator" {
				break
			}
			return err
		}

		var chancecard, c ChanceCard
		doc.DataTo(&c)
		chancecard.DocRef = doc.Ref.ID
		chancecard.Message = c.Message
		chancecard.IsPosChange = c.IsPosChange
		chancecard.IsScoreChange = c.IsScoreChange
		chancecard.IsToFieldMove = c.IsToFieldMove
		chancecard.Value = c.Value

		if len(ChanceCards) > i {
			ChanceCards[i] = chancecard
		} else {
			ChanceCards = append(ChanceCards, chancecard)
		}

		i++
	}

	return nil
}

// Edit Question
func EditChanceCard(ctx context.Context, client *firestore.Client, DocRef, Field, NewMessage string, NewValue int, NewboolValue bool) error {
	for i := range ChanceCards {
		if ChanceCards[i].DocRef == DocRef {
			switch Field {

			case "Message":
				ChanceCards[i].Message = NewMessage
				//Create an array of updates
				updates := []firestore.Update{
					{Path: "message", Value: NewMessage},
				}

				//Apply the updates
				_, err := client.Collection("tac").Doc(DocRef).Update(ctx, updates)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						return fmt.Errorf("chance card %s not found", DocRef)
					}
					return fmt.Errorf("error updating chance card: %v", err)
				}

			case "IsPosChange":
				ChanceCards[i].IsPosChange = NewboolValue
				//Create an array of updates
				updates := []firestore.Update{
					{Path: "isposchange", Value: NewboolValue},
				}

				//Apply the updates
				_, err := client.Collection("tac").Doc(DocRef).Update(ctx, updates)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						return fmt.Errorf("chance card %s not found", DocRef)
					}
					return fmt.Errorf("error updating chance card: %v", err)
				}

			case "IsScoreChange":
				ChanceCards[i].IsScoreChange = NewboolValue
				//Create an array of updates
				updates := []firestore.Update{
					{Path: "isscorechange", Value: NewboolValue},
				}

				//Apply the updates
				_, err := client.Collection("tac").Doc(DocRef).Update(ctx, updates)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						return fmt.Errorf("chance card %s not found", DocRef)
					}
					return fmt.Errorf("error updating chance card: %v", err)
				}

			case "IsToFieldChange":
				ChanceCards[i].IsToFieldMove = NewboolValue
				//Create an array of updates
				updates := []firestore.Update{
					{Path: "istofieldmove", Value: NewboolValue},
				}

				//Apply the updates
				_, err := client.Collection("tac").Doc(DocRef).Update(ctx, updates)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						return fmt.Errorf("chance card %s not found", DocRef)
					}
					return fmt.Errorf("error updating chance card: %v", err)
				}

			case "Value":
				ChanceCards[i].Value = NewValue
				//Create an array of updates
				updates := []firestore.Update{
					{Path: "value", Value: NewValue},
				}

				//Apply the updates
				_, err := client.Collection("tac").Doc(DocRef).Update(ctx, updates)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						return fmt.Errorf("chance card %s not found", DocRef)
					}
					return fmt.Errorf("error updating chance card: %v", err)
				}
			}

			break
		}
	}
	return nil
}

//Delete
func DelChanceCard(ctx context.Context, client *firestore.Client, DocRef string) error {
	_, err := client.Collection("tac").Doc(DocRef).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("chance card %s not found", DocRef)
		}
		return fmt.Errorf("error deleting chance card: %v", err)
	}

	i := 0
	var tempChanceCards []ChanceCard
	for {
		if i == len(ChanceCards) {
			return fmt.Errorf("chance card %s not found", DocRef)
		}

		if ChanceCards[i].DocRef == DocRef {
			tempChanceCards = append(tempChanceCards, ChanceCards[0:i]...)
			tempChanceCards = append(tempChanceCards, ChanceCards[i+1:]...)
			break
		}

		i++
	}

	ChanceCards = tempChanceCards

	return nil
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//Sessions
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//Create
func AddSession(ctx context.Context, firestoreClient *firestore.Client, UserDisplayName, Subject, Gamemode, Ai1, Ai2, Ai3, StartTime, EndTime, QDetails string, Score int64, CorrectAnswers, IncorrectAnswers int) error {
	data := map[string]interface{}{
		"userdisplayname":  UserDisplayName,
		"subject":          Subject,
		"gamemode":         Gamemode,
		"ai1":              Ai1,
		"ai2":              Ai2,
		"ai3":              Ai3,
		"starttime":        StartTime,
		"endtime":          EndTime,
		"score":            Score,
		"numcorrect":       CorrectAnswers,
		"numincorrect":     IncorrectAnswers,
		"questionsdetails": QDetails,
	}

	docRef, _, err := firestoreClient.Collection("sessions").Add(ctx, data)
	if err != nil {
		return err
	}

	var session Session
	session.DocRef = docRef.ID
	session.UserDisplayName = UserDisplayName
	session.Subject = Subject
	session.Gamemode = Gamemode
	session.Ai1 = Ai1
	session.Ai2 = Ai2
	session.Ai3 = Ai3
	session.StartTime = StartTime
	session.EndTime = EndTime
	session.Score = Score
	session.NumCorrect = CorrectAnswers
	session.NumIncorrect = IncorrectAnswers
	session.QDetails = QDetails

	Sessions = append(Sessions, session)

	return nil
}

//List
func GetSessions(ctx context.Context, firestoreClient *firestore.Client) error {
	i := 0
	iter := firestoreClient.Collection("sessions").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err != nil {
			if err.Error() == "no more items in iterator" {
				break
			}
			return err
		}

		var session, s Session
		doc.DataTo(&s)
		session.DocRef = doc.Ref.ID
		session.UserDisplayName = s.UserDisplayName
		session.Subject = s.Subject
		session.Gamemode = s.Gamemode
		session.Ai1 = s.Ai1
		session.Ai2 = s.Ai2
		session.Ai3 = s.Ai3
		session.StartTime = s.StartTime
		session.EndTime = s.EndTime
		session.Score = s.Score
		session.NumCorrect = s.NumCorrect
		session.NumIncorrect = s.NumIncorrect
		session.QDetails = s.QDetails

		if len(Sessions) > i {
			Sessions[i] = session
		} else {
			Sessions = append(Sessions, session)
		}

		i++
	}

	return nil
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//Images
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//Get Default Iamges
