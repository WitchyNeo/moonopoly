package MBSEGames

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Structures
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
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
	CorrectAnswers   int
	IncorrectAnswers int
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

type userTokenSource struct {
	AccessToken string
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//Global Variables & Constants
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
var Adjectives = [...]string{"able", "afraid", "angry", "awake", "bad", "beautiful", "big", "bitter", "black", "blue", "bold", "bored", "brave", "bright", "broad", "broken", "brown", "busy", "calm", "certain", "cheap", "clean", "clear", "clever", "close", "cloudy", "cold", "cool", "crazy", "cruel", "dark", "dead", "deep", "difficult", "dirty", "dry", "dull", "early", "easy", "empty", "equal", "excited", "faint", "false", "famous", "far", "fast", "fat", "few", "fine", "firm", "first", "flat", "foreign", "free", "fresh", "full", "funny", "gentle", "glad", "gold", "good", "gray", "great", "green", "happy", "hard", "heavy", "high", "hollow", "hot", "huge", "hungry", "ill", "important", "kind", "large", "last", "late", "lazy", "left", "light", "little", "lively", "lonely", "long", "loud", "low", "mad", "main", "many", "narrow", "near", "neat", "necessary", "new", "next", "nice", "noisy", "normal", "old", "open", "orange", "other", "pale", "past", "pink", "plain", "pleasant", "poor", "polite", "possible", "present", "pretty", "proud", "public", "purple", "quick", "quiet", "rare", "ready", "real", "red", "rich", "right", "rough", "round", "sad", "safe", "salty", "same", "scared", "secret", "serious", "sharp", "short", "shy", "sick", "silent", "silly", "simple", "single", "slow", "small", "smooth", "soft", "solid", "sour", "special", "square", "steady", "steep", "stiff", "strange", "strict", "strong", "stupid", "sudden", "sure", "sweet", "tall", "tame", "tender", "thick", "thin", "thirsty", "tidy", "tight", "tiny", "tired", "true", "ugly", "useful", "usual", "vague", "violet", "warm", "weak", "wet", "white", "whole", "wicked", "wide", "wild", "wise", "wonderful", "wrong", "yellow", "young"}
var Nouns = [...]string{"air", "animal", "answer", "apple", "area", "arm", "art", "baby", "back", "bag", "ball", "bank", "bed", "bird", "blood", "boat", "body", "bone", "book", "bottom", "box", "boy", "branch", "bread", "brother", "building", "bus", "car", "card", "case", "cat", "cause", "chair", "child", "city", "class", "cloud", "coat", "color", "company", "corner", "country", "course", "cow", "crowd", "cup", "day", "desk", "dog", "door", "dream", "dress", "ear", "earth", "egg", "end", "eye", "face", "fact", "family", "father", "fear", "field", "figure", "finger", "fire", "fish", "floor", "flower", "food", "foot", "force", "forest", "friend", "front", "fruit", "game", "garden", "girl", "glass", "gold", "government", "grass", "ground", "group", "hair", "hand", "hat", "head", "health", "heart", "hill", "home", "horse", "hospital", "hour", "house", "ice", "idea", "image", "industry", "island", "job", "key", "kind", "king", "kitchen", "lake", "land", "language", "law", "leaf", "leg", "letter", "level", "life", "light", "line", "list", "love", "machine", "man", "map", "market", "material", "matter", "meal", "meat", "member", "message", "metal", "milk", "mind", "minute", "money", "month", "moon", "morning", "mother", "mountain", "mouth", "music", "name", "nation", "neck", "needle", "neighbor", "network", "news", "night", "noise", "north", "nose", "note", "number", "ocean", "office", "oil", "order", "page", "paint", "paper", "parent", "park", "part", "party", "past", "path", "pen", "pencil", "people", "person", "phone", "picture", "piece", "pig", "place", "plan", "plane", "plant", "plate", "point", "police", "pool", "power", "president", "price", "problem", "product", "program", "property", "question", "radio", "rain", "reason", "record", "report", "rest", "result", "rice", "river", "road", "rock", "room", "root", "rule", "salt", "sand", "school", "science", "sea", "seat", "second", "seed", "sentence", "service", "shape", "sheep", "ship", "shirt", "shoe", "shop", "show", "side", "sign", "silver", "sister", "size", "skin", "sky", "snow", "soap", "society", "soil", "son", "song", "sound", "soup", "south", "space", "spoon", "sport", "spring", "square", "star", "state", "station", "steam", "steel", "step", "stick", "stone", "stop", "store", "story", "street", "student", "study", "sugar", "summer", "sun", "system", "table", "tail", "tea", "teacher", "team", "television", "test", "thing", "thought", "time", "toe", "top", "town", "toy", "track", "trade", "train", "tree", "trip", "trouble", "truck", "turn", "tv", "uncle", "unit", "valley", "value", "video", "view", "village", "voice", "wall", "war", "water", "way", "week", "weight", "west", "wheel", "wind", "window", "wine", "winter", "wire", "woman", "wood", "word", "work", "world", "yard", "year"}

const (
	WebApiKey = //Removed For Security
	ProjectID = //Removed For Security
)
var CurrentUser User

var Users []User
var Questions []Question
var ChanceCards []ChanceCard
var Sessions []Session

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//Intialisation
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func (t *userTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{
		AccessToken: t.AccessToken,
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(10 * time.Hour), 
	}, nil
}

func InitializeFirebaseWithUser(ctx context.Context, idToken string, projectID string) (*firebase.App, *firestore.Client, error) {
	
    // Create the "Bridge"
	tokenSource := &userTokenSource{AccessToken: idToken}
	
	conf := &firebase.Config{ProjectID: projectID}
	opt := option.WithTokenSource(tokenSource)

	// Initialize App
	app, err := firebase.NewApp(ctx, conf, opt)
	if err != nil {
		return nil, nil, fmt.Errorf("error initializing app: %w", err)
	}

	// Initialize Firestore (Authenticated as the User)
	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting Firestore client: %w", err)
	}

	return app, client, nil
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//Users
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//Login
func Login(ctx context.Context, app *firebase.App, firestoreClient *firestore.Client, email string) error {
	docGrab := firestoreClient.Collection("users").Where("email", "==", email).Documents(ctx)
	for {
		doc, err := docGrab.Next()
		if err != nil {
			if err.Error() == "no more items in iterator" {
				break
			}
			return err
		}

		var u User
		doc.DataTo(&u)

		CurrentUser.DocRef = doc.Ref.ID
		CurrentUser.UID = doc.Ref.ID
		CurrentUser.FirstName = u.FirstName
		CurrentUser.Surname = u.Surname
		CurrentUser.Email = email
		CurrentUser.CelNum = u.CelNum
		CurrentUser.DisplayName = u.DisplayName
		CurrentUser.PicID = u.PicID
		CurrentUser.Streak = u.Streak
		CurrentUser.TotalScore = u.TotalScore
		CurrentUser.TotalTime = u.TotalTime
		CurrentUser.RecordStreak = u.RecordStreak
		CurrentUser.IsAdmin = u.IsAdmin
		CurrentUser.LastPlayed = u.LastPlayed
	}

	err := GetQuestions(ctx, firestoreClient)
	if err != nil {
		return err
	}

	err = GetChanceCards(ctx, firestoreClient)
	if err != nil {
		return err
	}

	err = GetUsers(ctx, firestoreClient)
	if err != nil {
		return err
	}

	return nil

}

// List
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

// Edit User
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

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Questions
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// List
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

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Chance Cards
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// List
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
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Sessions
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Create
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
	session.CorrectAnswers = CorrectAnswers
	session.IncorrectAnswers = IncorrectAnswers
	session.QDetails = QDetails

	Sessions = append(Sessions, session)

	return nil
}
