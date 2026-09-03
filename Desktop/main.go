package main

import (
	"context"
	"embed"
	"encoding/csv"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io/fs"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	MBSEStats "MBSEGames/Code/Backend"
	MBSEGames "MBSEGames/Code/Backend/Database"

	"cloud.google.com/go/firestore"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tree"
	firebase "firebase.google.com/go/v4" // Added Firebase Admin SDK import
	"firebase.google.com/go/v4/auth"     // Added Firebase Auth import
	"github.com/zserge/lorca"
)

//go:embed Code/FirebaseAuth/login.html Code/FirebaseAuth/login.js Assets/Images/Moonopoly.png Assets/Images/DiceFace1.svg Assets/Images/DiceFace2.svg Assets/Images/DiceFace3.svg Assets/Images/DiceFace4.svg Assets/Images/DiceFace5.svg Assets/Images/DiceFace6.svg Assets/Images/BackgroundV3.png
var embeddedFiles embed.FS

// Global variable to store the received token
var receivedIDToken, currentUserID string
var loginWaitGroup sync.WaitGroup   // Used to signal login completion
var firebaseAuthClient *auth.Client // Added global for Firebase Auth client
var firestoreClient *firestore.Client
var app *firebase.App
var playerScore, intplayerPos, intai1Pos, intai2Pos, intai3Pos int
var strPlayerScoreLbl string
var playerMiss bool

type AI struct {
	Name          string
	StatsSection1 int
	StatsSection2 int
	StatsSection3 int
	StatsSection4 int
	StatsSection5 int
}

type tableQuestion struct {
	DocRef      string `table:"-"`
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

type tableChanceCard struct {
	DocRef        string `table:"-"`
	Message       string
	IsPosChange   bool
	IsScoreChange bool
	IsToFieldMove bool
	Value         int
}

type scoreLead struct {
	DisplayName string
	Score       int64
}

type streakLead struct {
	DisplayName string
	Streak      int
}

type timeLead struct {
	DisplayName string
	Time        int64
}

type disLead struct {
	DisplayName string
	Value       string
}

type tableUser struct {
	DisplayName   string
	IsAdmin       bool
	TotCorrect    int
	TotIncorrect  int
	TotTime       int64
	BestStreak    int
	CurrentStreak int
	NumSessions   int
}

type smSStatsDisplay struct {
	SRef string `display:"-"`
	PDName string `label:"Player DisplayName"`
	Subject string
	Difficulty string
	StartTime string
}

type smPStatsDisplay struct {
	PRef string `display:"-"`
	PDName string `label:"DisplayName"`
	AvgMin float64 `label:"Average Session Length(min)"`
	AvgAcc float64 `label:"Average Accuracy"`
	AvgCpS float64 `label:"Average Correct per Session"`
	AvgImp float64 `label:"Average Improvement"`
}

type smQStatsDisplay struct {
	QRef string `display:"-"`
	QTxt string `label:"Question Text"`
	CAns int `label:"Correct Answer"`
	NumAsk int `label:"Total Times Asked"`
	AnsAcc float64 `label:"Accuracy"`
}

type SesQData struct {
	QRef string `display:"-"`
	AnsGiven int `label:"Answer Given"`
	IsCorrect bool `label:"Correct"`
}

type SessionStats struct {
	SRef string
	PlayerDN string
	TimePlayed int
	Accuracy float64
	AvgQperMin float64
	QStats []SesQData
}

var SSel, QSel, PSel int
var SessStats []smSStatsDisplay
var PlayStats []smPStatsDisplay
var QuesStats []smQStatsDisplay
var SStats []SessionStats
var ais []AI
var Session MBSEGames.Session

// --- Main Application Logic ---

func main() {
	log.Println("Starting Application (Lorca Version)...")
	ctx := context.Background()
	firebaseapp, client, err := MBSEGames.InitializeFirebaseApp(ctx, "Code/m-b-s-e-moonpoly-v2-0-0-zutmic-firebase-adminsdk-h343l-b350bceeb2.json")

	if err != nil {
		log.Fatalf("Failed to initialize Firebase Admin SDK: %v", err)
	}

	firebaseAuthClient = client
	app = firebaseapp
	// Start the local server to serve embedded files
	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel() // Ensure server context is eventually cancelled

	listener, serverAddr, err := startLocalServer(serverCtx, embeddedFiles)
	if err != nil {
		log.Fatalf("Failed to start local server: %v", err)
	}
	log.Printf("Local server listening on: %s\n", serverAddr)

	// --- Trigger Login Flow ---
	loginWaitGroup.Add(1)          // Expect one signal for login completion
	go showLoginWindow(serverAddr) // Launch Lorca UI in a separate goroutine

	// --- Wait for Login ---
	log.Println("Waiting for login completion...")
	loginWaitGroup.Wait() // Block here until handleTokenFromJS signals done

	// --- Post-Login ---
	if receivedIDToken != "" {
		handleLoginSuccess(firebaseAuthClient)
	} else {
		log.Println("Login process completed, but no token was received.")
		// This might happen if the user closed the window before logging in
	}

	// --- Continue Main App Logic ---
	firestoreClient, err = MBSEGames.GetFirestoreClient(ctx, app)
	if err != nil {
		log.Printf("firestore client error: %s", err)
	}

	err = MBSEGames.Login(ctx, app, firestoreClient, currentUserID)
	if err != nil {
		log.Printf("error finding user '%s', error message: %v", currentUserID, err)
	}

	CurrentUser := MBSEGames.CurrentUser
	CUserName := CurrentUser.DisplayName

	var Users []MBSEGames.User
	var Sessions []MBSEGames.Session
	var Questions []MBSEGames.Question
	var lQs, ccc, mmm, lbl, lol, ttt, de, co []MBSEGames.Question
	var ChanceCards []MBSEGames.ChanceCard

	err = MBSEGames.GetUsers(ctx, firestoreClient)
	if err != nil {
		log.Printf("error getting Clients: %s", err)
	}

	if CurrentUser.IsAdmin {
		Sessions = append(Sessions, MBSEGames.Sessions...)
	}

	err = MBSEGames.GetQuestions(ctx, firestoreClient)
	if err != nil {
		log.Printf("error getting Questions: %s", err)
	}

	err = MBSEGames.GetChanceCards(ctx, firestoreClient)
	if err != nil {
		log.Printf("error getting Chance Cards: %s", err)
	}

	Questions = append(Questions, MBSEGames.Questions...)
	ChanceCards = append(ChanceCards, MBSEGames.ChanceCards...)
	Users = append(Users, MBSEGames.Users...)

	var ai AI
	ai.Name = "Numinor"
	ai.StatsSection1 = 60
	ai.StatsSection2 = 60
	ai.StatsSection3 = 60
	ai.StatsSection4 = 60
	ai.StatsSection5 = 60
	ais = append(ais, ai)

	ai.Name = "Numinor"
	ai.StatsSection1 = 70
	ai.StatsSection2 = 70
	ai.StatsSection3 = 70
	ai.StatsSection4 = 70
	ai.StatsSection5 = 70
	ais = append(ais, ai)

	ai.Name = "Numinor"
	ai.StatsSection1 = 80
	ai.StatsSection2 = 80
	ai.StatsSection3 = 80
	ai.StatsSection4 = 80
	ai.StatsSection5 = 80
	ais = append(ais, ai)

	ai.Name = "Samuel"
	ai.StatsSection1 = 70
	ai.StatsSection2 = 55
	ai.StatsSection3 = 55
	ai.StatsSection4 = 55
	ai.StatsSection5 = 55
	ais = append(ais, ai)

	ai.Name = "Samuel"
	ai.StatsSection1 = 80
	ai.StatsSection2 = 65
	ai.StatsSection3 = 65
	ai.StatsSection4 = 65
	ai.StatsSection5 = 65
	ais = append(ais, ai)

	ai.Name = "Samuel"
	ai.StatsSection1 = 90
	ai.StatsSection2 = 75
	ai.StatsSection3 = 75
	ai.StatsSection4 = 75
	ai.StatsSection5 = 75
	ais = append(ais, ai)

	ai.Name = "Ajani"
	ai.StatsSection1 = 55
	ai.StatsSection2 = 70
	ai.StatsSection3 = 55
	ai.StatsSection4 = 55
	ai.StatsSection5 = 55
	ais = append(ais, ai)

	ai.Name = "Ajani"
	ai.StatsSection1 = 65
	ai.StatsSection2 = 80
	ai.StatsSection3 = 65
	ai.StatsSection4 = 65
	ai.StatsSection5 = 65
	ais = append(ais, ai)

	ai.Name = "Ajani"
	ai.StatsSection1 = 75
	ai.StatsSection2 = 90
	ai.StatsSection3 = 75
	ai.StatsSection4 = 75
	ai.StatsSection5 = 75
	ais = append(ais, ai)

	ai.Name = "Trell"
	ai.StatsSection1 = 55
	ai.StatsSection2 = 55
	ai.StatsSection3 = 70
	ai.StatsSection4 = 55
	ai.StatsSection5 = 55
	ais = append(ais, ai)

	ai.Name = "Trell"
	ai.StatsSection1 = 65
	ai.StatsSection2 = 65
	ai.StatsSection3 = 80
	ai.StatsSection4 = 65
	ai.StatsSection5 = 65
	ais = append(ais, ai)

	ai.Name = "Trell"
	ai.StatsSection1 = 75
	ai.StatsSection2 = 75
	ai.StatsSection3 = 90
	ai.StatsSection4 = 75
	ai.StatsSection5 = 75
	ais = append(ais, ai)

	ai.Name = "Quintin"
	ai.StatsSection1 = 55
	ai.StatsSection2 = 55
	ai.StatsSection3 = 55
	ai.StatsSection4 = 70
	ai.StatsSection5 = 55
	ais = append(ais, ai)

	ai.Name = "Quintin"
	ai.StatsSection1 = 65
	ai.StatsSection2 = 65
	ai.StatsSection3 = 65
	ai.StatsSection4 = 80
	ai.StatsSection5 = 65
	ais = append(ais, ai)

	ai.Name = "Quintin"
	ai.StatsSection1 = 75
	ai.StatsSection2 = 75
	ai.StatsSection3 = 75
	ai.StatsSection4 = 80
	ai.StatsSection5 = 75
	ais = append(ais, ai)

	ai.Name = "Quasar"
	ai.StatsSection1 = 55
	ai.StatsSection2 = 55
	ai.StatsSection3 = 55
	ai.StatsSection4 = 55
	ai.StatsSection5 = 80
	ais = append(ais, ai)

	ai.Name = "Quasar"
	ai.StatsSection1 = 65
	ai.StatsSection2 = 65
	ai.StatsSection3 = 65
	ai.StatsSection4 = 65
	ai.StatsSection5 = 90
	ais = append(ais, ai)

	ai.Name = "Quasar"
	ai.StatsSection1 = 75
	ai.StatsSection2 = 75
	ai.StatsSection3 = 75
	ai.StatsSection4 = 75
	ai.StatsSection5 = 100
	ais = append(ais, ai)

	var Ai1, Ai2, Ai3 int
	var GameMode string
	var startTime time.Time

	var leadScores []scoreLead
	var leadStreaks []streakLead
	var leadTimes []timeLead
	for _, u := range Users {
		leadScores = append(leadScores, scoreLead{u.DisplayName, u.TotalScore})
		leadStreaks = append(leadStreaks, streakLead{u.DisplayName, u.Streak})
		leadTimes = append(leadTimes, timeLead{u.DisplayName, u.TotalTime})
	}
	sort.Slice(leadScores, func(i, j int) bool {
		return leadScores[i].Score > leadScores[j].Score
	})
	sort.Slice(leadStreaks, func(i, j int) bool {
		return leadStreaks[i].Streak > leadStreaks[j].Streak
	})
	sort.Slice(leadTimes, func(i, j int) bool {
		return leadTimes[i].Time > leadTimes[j].Time
	})

	var sec, min, hour, days int
	var disTL, disScL, disStL []disLead
	for i := 0; i < 7; i++ {
		totNanSec := int(leadTimes[i].Time) / 1000000000
		sec = totNanSec % 60
		nsTime := (totNanSec - sec) / 60
		min = nsTime % 60
		nmTime := (nsTime - min) / 60
		hour = nmTime % 24
		days = (nmTime - hour) / 24

		var strTime string
		if days > 0 {
			strTime = "D: " + strconv.Itoa(days)
		} else if hour > 0 {
			strTime = "H: " + strconv.Itoa(hour)
		} else if min > 0 {
			strTime = "M: " + strconv.Itoa(min)
		} else {
			strTime = "S: " + strconv.Itoa(sec)
		}

		var tempdis disLead
		tempdis.DisplayName = leadTimes[i].DisplayName
		tempdis.Value = strTime
		if len(disTL) < 7 {
			disTL = append(disTL, tempdis)
		} else {
			disTL[i] = tempdis
		}

		tempdis.DisplayName = leadScores[i].DisplayName
		tempdis.Value = strconv.FormatInt(leadScores[i].Score, 10)
		if len(disScL) < 7 {
			disScL = append(disScL, tempdis)
		} else {
			disScL[i] = tempdis
		}

		tempdis.DisplayName = leadStreaks[i].DisplayName
		tempdis.Value = strconv.Itoa(leadStreaks[i].Streak)
		if len(disStL) < 7 {
			disStL = append(disStL, tempdis)
		} else {
			disStL[i] = tempdis
		}

	}

	b := core.NewBody()
	b.Scene.SetFullscreen(true)
	pagesWidget := core.NewPages(b)

	var tableQuestions []tableQuestion
	var tableChanceCards []tableChanceCard
	var tableUsers []tableUser
	if CurrentUser.IsAdmin {
		pagesWidget.Open("Admin")

		for i := range Questions {
			var q tableQuestion
			q.DocRef = Questions[i].DocRef
			q.Subject = Questions[i].Subject
			q.Category = Questions[i].Category
			q.Outcome = Questions[i].Outcome
			q.Level = Questions[i].Level
			q.Question = Questions[i].Question
			q.Option1 = Questions[i].Option1
			q.Option2 = Questions[i].Option2
			q.Option3 = Questions[i].Option3
			q.Option4 = Questions[i].Option4
			q.Answer = Questions[i].Answer
			q.Explanation = Questions[i].Explanation
			tableQuestions = append(tableQuestions, q)
		}

		for i := range ChanceCards {
			var cc tableChanceCard
			cc.DocRef = ChanceCards[i].DocRef
			cc.Message = ChanceCards[i].Message
			cc.IsPosChange = ChanceCards[i].IsPosChange
			cc.IsScoreChange = ChanceCards[i].IsScoreChange
			cc.IsToFieldMove = ChanceCards[i].IsToFieldMove
			cc.Value = ChanceCards[i].Value
			tableChanceCards = append(tableChanceCards, cc)
		}
		
	} else {
		pagesWidget.Open("Player")
	}

	/////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	//////Admin Page
	/////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	pagesWidget.AddPage("Admin", func(pg *core.Pages) {
		frAdminHome := core.NewFrame(pg)
		frAdminHome.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.CenterAll()
			s.Grow.Set(1, 1)
		})

		//TopAdminFrame
		frTopAdmin := core.NewFrame(frAdminHome)
		frTopAdmin.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
			s.CenterAll()
			s.Gap.Set(units.Vw(40))
		})

		//Admin ManQuestions Page
		btnQuestions := core.NewButton(frTopAdmin).SetText("Questions").SetIcon(icons.QuestionMarkFill)
		btnQuestions.Styler(func(s *styles.Style) {
			s.Min.Set(units.Vw(20), units.Vh(20))
			s.Max.Set(units.Vw(20), units.Vh(20))
		})
		btnQuestions.OnClick(func(e events.Event) {
			winQuestions := core.NewBody("Manage Questions")

			var intSelQIndex int
			tblQuestions := core.NewTable(winQuestions).SetSlice(&tableQuestions).BindSelect(&intSelQIndex)
			tbQuestions := core.NewToolbar(winQuestions)
			tbQuestions.Maker(func(p *tree.Plan) {
				tree.Add(p, func(w *core.Button) {
					w.SetIcon(icons.AddFill)
					w.OnClick(func(e events.Event) {
						diaAddQuestion := core.NewBody("Add Question")
						diaAddQuestion.Styler(func(s *styles.Style) {
							s.Direction.SetString("Column")
						})
						//Subject
						core.NewText(diaAddQuestion).SetText("Subject:")
						txtSubject := core.NewTextField(diaAddQuestion)

						//Category
						core.NewText(diaAddQuestion).SetText("Category:")
						txtCategory := core.NewTextField(diaAddQuestion)

						//Outcome
						core.NewText(diaAddQuestion).SetText("Outcome:")
						spnOutcome := core.NewSpinner(diaAddQuestion)
						spnOutcome.SetMin(100)
						spnOutcome.SetMax(400)
						spnOutcome.SetStep(100).SetEnforceStep(true)

						//Level
						core.NewText(diaAddQuestion).SetText("Level:")
						spnLevel := core.NewSpinner(diaAddQuestion)
						spnLevel.SetMin(1)
						spnLevel.SetMax(3)
						spnLevel.SetStep(1).SetEnforceStep(true)

						//Question
						core.NewText(diaAddQuestion).SetText("Question:")
						txtQuestion := core.NewTextField(diaAddQuestion)

						//Option1
						core.NewText(diaAddQuestion).SetText("Option 1:")
						txtOption1 := core.NewTextField(diaAddQuestion)

						//Option2
						core.NewText(diaAddQuestion).SetText("Option 2:")
						txtOption2 := core.NewTextField(diaAddQuestion)

						//Option3
						core.NewText(diaAddQuestion).SetText("Option 3:")
						txtOption3 := core.NewTextField(diaAddQuestion)

						//Option4
						core.NewText(diaAddQuestion).SetText("Option 4:")
						txtOption4 := core.NewTextField(diaAddQuestion)

						//Answer
						core.NewText(diaAddQuestion).SetText("Correct answer:")
						spnAnswer := core.NewSpinner(diaAddQuestion)
						spnAnswer.SetMin(1)
						spnAnswer.SetMax(4)
						spnAnswer.SetStep(1).SetEnforceStep(true)

						//Explanation
						core.NewText(diaAddQuestion).SetText("Explanation:")
						txtExplanation := core.NewTextField(diaAddQuestion)

						diaAddQuestion.AddBottomBar(func(bar *core.Frame) {
							diaAddQuestion.AddCancel(bar).OnClick(func(e events.Event) {
								core.MessageSnackbar(w, "Question Not Added")
							})
							diaAddQuestion.AddOK(bar).OnClick(func(e events.Event) {
								MBSEGames.AddQuestion(ctx, firestoreClient, txtSubject.Text(), txtCategory.Text(), txtQuestion.Text(), txtOption1.Text(), txtOption2.Text(), txtOption3.Text(), txtOption4.Text(), txtExplanation.Text(), int(spnOutcome.Value), int(spnLevel.Value), int(spnAnswer.Value))
								core.MessageSnackbar(w, "Question Added Successfully!")
							})
						})
						diaAddQuestion.RunDialog(w)
					})
				})

				tree.Add(p, func(w *core.FileButton) {
					w.OnChange(func(e events.Event) {
						QuestFilepath := w.Filename
						file, err := os.Open(QuestFilepath)
						if err != nil {
							log.Fatal(err)
						}
						r := csv.NewReader(file)
						qs, err := r.ReadAll()
						if err != nil {
							log.Fatal(err)
						}

						for i, q := range qs {
							if i != 0 {
								q2, err := strconv.Atoi(q[2])
								if err != nil {
									log.Fatal(err)
								}
								q3, err := strconv.Atoi(q[3])
								if err != nil {
									log.Fatal(err)
								}
								var q9 int
								switch q[9]{
								case "A": q9 = 1
								case "B": q9 = 2
								case "C": q9 = 3
								case "D": q9 = 4
								default:
									q9, err = strconv.Atoi(q[9])
									if err != nil {
										log.Fatal(err)
									}
								}
								err = MBSEGames.AddQuestion(ctx, firestoreClient, q[0], q[1], q[4], q[5], q[6], q[7], q[8], q[10], q2, q3, q9)
								if err != nil {
									log.Fatal(err)
								}

								var tq tableQuestion
								tq.Subject = q[0]
								tq.Category = q[1]
								tq.Question = q[4]
								tq.Option1 = q[5]
								tq.Option2 = q[6]
								tq.Option3 = q[7]
								tq.Option4 = q[8]
								tq.Explanation = q[10]
								tq.Outcome = q2
								tq.Level = q3
								tq.Answer = q9

								tableQuestions = append(tableQuestions, tq)
							}
						} 
						tblQuestions.Update()
					})
				})

				tree.Add(p, func(w *core.Button) {
					w.SetIcon(icons.EditFill)
					w.OnClick(func(e events.Event) {
						diaEditQuestion := core.NewBody("Edit Question")
						diaEditQuestion.Styler(func(s *styles.Style) {
							s.Direction.SetString("Column")
						})
						//Subject
						core.NewText(diaEditQuestion).SetText("Subject:")
						txtSubject := core.NewTextField(diaEditQuestion).SetPlaceholder(Questions[tblQuestions.SelectedIndex].Subject)

						//Category
						core.NewText(diaEditQuestion).SetText("Category:")
						txtCategory := core.NewTextField(diaEditQuestion).SetPlaceholder(Questions[tblQuestions.SelectedIndex].Category)

						//Outcome
						core.NewText(diaEditQuestion).SetText("Outcome:")
						spnOutcome := core.NewSpinner(diaEditQuestion)
						spnOutcome.SetMin(100)
						spnOutcome.SetMax(400)
						spnOutcome.SetStep(100).SetEnforceStep(true).SetValue(float32(Questions[tblQuestions.SelectedIndex].Outcome))

						//Level
						core.NewText(diaEditQuestion).SetText("Level:")
						spnLevel := core.NewSpinner(diaEditQuestion)
						spnLevel.SetMin(1)
						spnLevel.SetMax(3)
						spnLevel.SetStep(1).SetEnforceStep(true).SetValue(float32(Questions[tblQuestions.SelectedIndex].Level))

						//Question
						core.NewText(diaEditQuestion).SetText("Question:")
						txtQuestion := core.NewTextField(diaEditQuestion).SetPlaceholder(Questions[tblQuestions.SelectedIndex].Question)

						//Option1
						core.NewText(diaEditQuestion).SetText("Option 1:")
						txtOption1 := core.NewTextField(diaEditQuestion).SetPlaceholder(Questions[tblQuestions.SelectedIndex].Option1)

						//Option2
						core.NewText(diaEditQuestion).SetText("Option 2:")
						txtOption2 := core.NewTextField(diaEditQuestion).SetPlaceholder(Questions[tblQuestions.SelectedIndex].Option2)

						//Option3
						core.NewText(diaEditQuestion).SetText("Option 3:")
						txtOption3 := core.NewTextField(diaEditQuestion).SetPlaceholder(Questions[tblQuestions.SelectedIndex].Option3)

						//Option4
						core.NewText(diaEditQuestion).SetText("Option 4:")
						txtOption4 := core.NewTextField(diaEditQuestion).SetPlaceholder(Questions[tblQuestions.SelectedIndex].Option4)

						//Answer
						core.NewText(diaEditQuestion).SetText("Correct answer:")
						spnAnswer := core.NewSpinner(diaEditQuestion)
						spnAnswer.SetMin(1)
						spnAnswer.SetMax(4)
						spnAnswer.SetStep(1).SetEnforceStep(true).SetValue(float32(Questions[tblQuestions.SelectedIndex].Answer))

						//Explanation
						core.NewText(diaEditQuestion).SetText("Explanation:")
						txtExplanation := core.NewTextField(diaEditQuestion).SetPlaceholder(Questions[tblQuestions.SelectedIndex].Explanation)

						diaEditQuestion.AddBottomBar(func(bar *core.Frame) {
							diaEditQuestion.AddCancel(bar).OnClick(func(e events.Event) {
								core.MessageSnackbar(w, "Question Not Edited")
							})
							diaEditQuestion.AddOK(bar).OnClick(func(e events.Event) {
								if txtSubject.Text() != Questions[tblQuestions.SelectedIndex].Subject {
									MBSEGames.EditQuestion(ctx, firestoreClient, Questions[tblQuestions.SelectedIndex].DocRef, "Subject", txtSubject.Text(), 0, 0)
								}

								if txtCategory.Text() != Questions[tblQuestions.SelectedIndex].Category {
									MBSEGames.EditQuestion(ctx, firestoreClient, Questions[tblQuestions.SelectedIndex].DocRef, "Category", txtCategory.Text(), 0, 0)
								}

								if spnOutcome.Value != float32(Questions[tblQuestions.SelectedIndex].Outcome) {
									MBSEGames.EditQuestion(ctx, firestoreClient, Questions[tblQuestions.SelectedIndex].DocRef, "Outcome", "", int(spnOutcome.Value), 0)
								}

								if spnLevel.Value != float32(Questions[tblQuestions.SelectedIndex].Level) {
									MBSEGames.EditQuestion(ctx, firestoreClient, Questions[tblQuestions.SelectedIndex].DocRef, "Level", "", int(spnLevel.Value), 0)
								}

								if txtQuestion.Text() != Questions[tblQuestions.SelectedIndex].Question {
									MBSEGames.EditQuestion(ctx, firestoreClient, Questions[tblQuestions.SelectedIndex].DocRef, "Question", txtQuestion.Text(), 0, 0)
								}

								if txtOption1.Text() != Questions[tblQuestions.SelectedIndex].Option1 {
									MBSEGames.EditQuestion(ctx, firestoreClient, Questions[tblQuestions.SelectedIndex].DocRef, "Option1", txtOption1.Text(), 0, 0)
								}

								if txtOption2.Text() != Questions[tblQuestions.SelectedIndex].Option2 {
									MBSEGames.EditQuestion(ctx, firestoreClient, Questions[tblQuestions.SelectedIndex].DocRef, "Option2", txtOption2.Text(), 0, 0)
								}

								if txtOption3.Text() != Questions[tblQuestions.SelectedIndex].Option3 {
									MBSEGames.EditQuestion(ctx, firestoreClient, Questions[tblQuestions.SelectedIndex].DocRef, "Option3", txtOption3.Text(), 0, 0)
								}

								if txtOption4.Text() != Questions[tblQuestions.SelectedIndex].Option4 {
									MBSEGames.EditQuestion(ctx, firestoreClient, Questions[tblQuestions.SelectedIndex].DocRef, "Option4", txtOption4.Text(), 0, 0)
								}

								if spnAnswer.Value != float32(Questions[tblQuestions.SelectedIndex].Answer) {
									MBSEGames.EditQuestion(ctx, firestoreClient, Questions[tblQuestions.SelectedIndex].DocRef, "Answer", "", int(spnAnswer.Value), 0)
								}

								if txtExplanation.Text() != Questions[tblQuestions.SelectedIndex].Explanation {
									MBSEGames.EditQuestion(ctx, firestoreClient, Questions[tblQuestions.SelectedIndex].DocRef, "Explanation", txtExplanation.Text(), 0, 0)
								}
								core.MessageSnackbar(w, "Question Edited Successfully!")
							})
						})
						diaEditQuestion.RunDialog(w)
					})
				})

				tree.Add(p, func(w *core.Button) {
					w.SetIcon(icons.DeleteFill)
					w.OnClick(func(e events.Event) {
						diaDelQuestion := core.NewBody("Delete Question")
						core.NewText(diaDelQuestion).SetText("Are you sure you want to DELETE this Question?")
						diaDelQuestion.AddBottomBar(func(bar *core.Frame) {
							diaDelQuestion.AddCancel(bar).OnClick(func(e events.Event) {
								core.MessageSnackbar(w, "Question Not Deleted")
							})
							diaDelQuestion.AddOK(bar).OnClick(func(e events.Event) {
								MBSEGames.DelQuestion(ctx, firestoreClient, Questions[tblQuestions.SelectedIndex].DocRef)
								core.MessageSnackbar(w, "Question Deleted")
							})
						})
						diaDelQuestion.RunDialog(w)
					})
				})
			})
			winQuestions.RunFullDialog(btnQuestions)
		})

		//Admin ManChanceCards Page
		btnChanceCards := core.NewButton(frTopAdmin).SetText("Chance Cards").SetIcon(icons.PlayingCardsFill)
		btnChanceCards.Styler(func(s *styles.Style) {
			s.Min.Set(units.Vw(20), units.Vh(20))
			s.Max.Set(units.Vw(20), units.Vh(20))
		})
		btnChanceCards.OnClick(func(e events.Event) {
			winChanceCards := core.NewBody("Manage Chance Cards")

			var intSelQIndex int
			tblChanceCards := core.NewTable(winChanceCards).SetSlice(&tableChanceCards).BindSelect(&intSelQIndex)
			tbChanceCards := core.NewToolbar(winChanceCards)
			tbChanceCards.Maker(func(p *tree.Plan) {
				tree.Add(p, func(w *core.Button) {
					w.SetIcon(icons.AddFill)
					w.OnClick(func(e events.Event) {
						diaAddChanceCard := core.NewBody("Add Chance Card")
						diaAddChanceCard.Styler(func(s *styles.Style) {
							s.Direction.SetString("Column")
						})
						//Message
						core.NewText(diaAddChanceCard).SetText("Message:")
						txtMessage := core.NewTextField(diaAddChanceCard)

						//IsPosChange
						core.NewText(diaAddChanceCard).SetText("Is Positive Change:")
						swPosChange := core.NewSwitch(diaAddChanceCard)

						//IsScoreChange
						core.NewText(diaAddChanceCard).SetText("Is Score Change:")
						swScoreChange := core.NewSwitch(diaAddChanceCard)

						//IsLMove
						core.NewText(diaAddChanceCard).SetText("Is Large Move:")
						swLMove := core.NewSwitch(diaAddChanceCard)

						//Value
						core.NewText(diaAddChanceCard).SetText("Value:")
						spnValue := core.NewSpinner(diaAddChanceCard)
						spnValue.SetMin(-600).SetMax(600).SetStep(1).SetEnforceStep(true)

						diaAddChanceCard.AddBottomBar(func(bar *core.Frame) {
							diaAddChanceCard.AddCancel(bar).OnClick(func(e events.Event) {
								core.MessageSnackbar(w, "Chance Card Not Added")
							})
							diaAddChanceCard.AddOK(bar).OnClick(func(e events.Event) {
								MBSEGames.AddChanceCard(ctx, firestoreClient, txtMessage.Text(), int(spnValue.Value), swPosChange.IsChecked(), swScoreChange.IsChecked(), swLMove.IsChecked())
								core.MessageSnackbar(w, "Chance Card Added Successfully!")
							})
						})
						diaAddChanceCard.RunDialog(w)
					})
				})

				tree.Add(p, func(w *core.FileButton) {
					w.OnChange(func(e events.Event) {
						CCFilepath := w.Filename
						byteRawCC, err := os.ReadFile(CCFilepath)
						if err != nil {
							log.Fatal(err)
						}

						strRawCC := string(byteRawCC)
						SplitCC := strings.Split(strRawCC, ",")

						var cc MBSEGames.ChanceCard
						i := 0
						for _, val := range SplitCC {
							i++
							switch i {
							case 1:
								cc.Message = val
							case 2:
								if val == "true" {
									cc.IsPosChange = true
								} else {
									cc.IsPosChange = false
								}
							case 3:
								if val == "true" {
									cc.IsScoreChange = true
								} else {
									cc.IsScoreChange = false
								}
							case 4:
								if val == "true" {
									cc.IsToFieldMove = true
								} else {
									cc.IsToFieldMove = false
								}
							case 5:
								cc.Value, err = strconv.Atoi(val)
								if err != nil {
									log.Fatal(err)
								}
								i = 0
								err = MBSEGames.AddChanceCard(ctx, firestoreClient, cc.Message, cc.Value, cc.IsPosChange, cc.IsScoreChange, cc.IsToFieldMove)
								if err != nil {
									log.Fatalf("Error Adding Chance Cards: %s", err)
								}

								for j, chancecard := range MBSEGames.ChanceCards {
									if len(tableChanceCards) <= j {
										var tcc tableChanceCard
										tcc.DocRef = chancecard.DocRef
										tcc.Message = chancecard.Message
										tcc.IsPosChange = chancecard.IsPosChange
										tcc.IsScoreChange = chancecard.IsScoreChange
										tcc.IsToFieldMove = chancecard.IsToFieldMove
										tcc.Value = chancecard.Value
										tableChanceCards = append(tableChanceCards, tcc)
									}
								}
							}
						}
						tblChanceCards.Update()
					})
				})

				tree.Add(p, func(w *core.Button) {
					w.SetIcon(icons.EditFill)
					w.OnClick(func(e events.Event) {
						diaEditChanceCard := core.NewBody("Edit Chance Card")
						diaEditChanceCard.Styler(func(s *styles.Style) {
							s.Direction.SetString("Column")
						})
						//Message
						core.NewText(diaEditChanceCard).SetText("Message:")
						txtMessage := core.NewTextField(diaEditChanceCard).SetPlaceholder(ChanceCards[tblChanceCards.SelectedIndex].Message)

						//IsPosChange
						core.NewText(diaEditChanceCard).SetText("Is Positive Change:")
						swPosChange := core.NewSwitch(diaEditChanceCard)
						if ChanceCards[tblChanceCards.SelectedIndex].IsPosChange {
							swPosChange.SetChecked(true)
						}

						//IsScoreChange
						core.NewText(diaEditChanceCard).SetText("Is Score Change:")
						swScoreChange := core.NewSwitch(diaEditChanceCard)
						if ChanceCards[tblChanceCards.SelectedIndex].IsScoreChange {
							swScoreChange.SetChecked(true)
						}

						//IsLMove
						core.NewText(diaEditChanceCard).SetText("Is Large Move:")
						swLMove := core.NewSwitch(diaEditChanceCard)
						if ChanceCards[tblChanceCards.SelectedIndex].IsToFieldMove {
							swLMove.SetChecked(true)
						}

						//Value
						core.NewText(diaEditChanceCard).SetText("Value:")
						spnValue := core.NewSpinner(diaEditChanceCard)
						spnValue.SetMin(-600).SetMax(600).SetStep(1).SetEnforceStep(true)

						diaEditChanceCard.AddBottomBar(func(bar *core.Frame) {
							diaEditChanceCard.AddCancel(bar).OnClick(func(e events.Event) {
								core.MessageSnackbar(w, "Chance Card Not Edited")
							})
							diaEditChanceCard.AddOK(bar).OnClick(func(e events.Event) {
								if txtMessage.Text() != ChanceCards[tblChanceCards.SelectedIndex].Message {
									MBSEGames.EditChanceCard(ctx, firestoreClient, ChanceCards[tblChanceCards.SelectedIndex].DocRef, "Message", txtMessage.Text(), 0, false)
								}

								if swPosChange.IsChecked() != ChanceCards[tblChanceCards.SelectedIndex].IsPosChange {
									MBSEGames.EditChanceCard(ctx, firestoreClient, ChanceCards[tblChanceCards.SelectedIndex].DocRef, "IsPosChange", "", 0, swPosChange.IsChecked())
								}

								if swScoreChange.IsChecked() != ChanceCards[tblChanceCards.SelectedIndex].IsScoreChange {
									MBSEGames.EditChanceCard(ctx, firestoreClient, ChanceCards[tblChanceCards.SelectedIndex].DocRef, "IsScoreChange", "", 0, swScoreChange.IsChecked())
								}

								if swLMove.IsChecked() != ChanceCards[tblChanceCards.SelectedIndex].IsToFieldMove {
									MBSEGames.EditChanceCard(ctx, firestoreClient, ChanceCards[tblChanceCards.SelectedIndex].DocRef, "IsLMove", "", 0, swLMove.IsChecked())
								}

								if spnValue.Value != float32(ChanceCards[tblChanceCards.SelectedIndex].Value) {
									MBSEGames.EditChanceCard(ctx, firestoreClient, ChanceCards[tblChanceCards.SelectedIndex].DocRef, "Value", "", int(spnValue.Value), false)
								}

								core.MessageSnackbar(w, "Chance Card Edited Successfully!")
							})
						})
						diaEditChanceCard.RunDialog(w)
					})
				})

				tree.Add(p, func(w *core.Button) {
					w.SetIcon(icons.DeleteFill)
					w.OnClick(func(e events.Event) {
						diaDelChanceCard := core.NewBody("Delete Chance Card")
						core.NewText(diaDelChanceCard).SetText("Are you sure you want to DELETE this Chance Card?")
						diaDelChanceCard.AddBottomBar(func(bar *core.Frame) {
							diaDelChanceCard.AddCancel(bar).OnClick(func(e events.Event) {
								core.MessageSnackbar(w, "Chance Card Not Deleted")
							})
							diaDelChanceCard.AddOK(bar).OnClick(func(e events.Event) {
								MBSEGames.DelChanceCard(ctx, firestoreClient, ChanceCards[tblChanceCards.SelectedIndex].DocRef)
								core.MessageSnackbar(w, "Chance Card Deleted")
							})
						})
						diaDelChanceCard.RunDialog(w)
					})
				})
			})
			winChanceCards.RunFullDialog(btnChanceCards)
		})

		//MidAdminFrame
		frMidAdmin := core.NewFrame(frAdminHome)
		frMidAdmin.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
			s.CenterAll()
		})

		//Admin Stats Page
		btnStatistics := core.NewButton(frMidAdmin).SetText("Stats").SetIcon(icons.QueryStatsFill)
		btnStatistics.Styler(func(s *styles.Style) {
			s.Min.Set(units.Vw(20), units.Vh(20))
			s.Max.Set(units.Vw(20), units.Vh(20))
		})
		btnStatistics.OnClick(func(e events.Event) {
			winStats := core.NewBody("Stats")
			MBSEGames.GetSessions(ctx, firestoreClient)
			MBSEStats.CalcStats()
			for i, sess := range MBSEStats.SStats {
				var s smSStatsDisplay
				s.SRef = sess.SRef 
				s.PDName = sess.PlayerDN
				s.Subject = MBSEStats.RawSessions[i].Subject
				s.Difficulty = MBSEStats.RawSessions[i].Gamemode
				s.StartTime = MBSEStats.RawSessions[i].StartTime

				SessStats = append(SessStats, s)
			}
			for _, play := range MBSEStats.Pstats {
				var p smPStatsDisplay
				p.PRef = play.PRef
				p.PDName = play.PlayerDN
				p.AvgCpS = play.AvgCSession
				p.AvgAcc = play.AvgASession
				p.AvgMin = play.AvgMinSession
				p.AvgImp = play.AvgImprovement

				PlayStats = append(PlayStats, p)
			}
			for _, ques := range MBSEStats.QStats {
				var q smQStatsDisplay
				q.QRef = ques.QRef
				for _, question := range Questions {
					if q.QRef == question.DocRef {
						q.QTxt = question.Question
						q.CAns = question.Answer
					}
				}
				q.NumAsk = ques.TotAsked
				q.AnsAcc = ques.TotAccuracy

				QuesStats = append(QuesStats, q)
			}

			frStatSum := core.NewFrame(winStats)
			frStatSum.Styler(func(s *styles.Style) {
				s.Direction = styles.Row
				s.Min.Set(units.Vh(100), units.Vw(100))
				s.Grow.Set(1, 1)
			})
			frSStatsMain := core.NewFrame(frStatSum)
			frSStatsMain.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
			})
			core.NewText(frSStatsMain).SetText("Average Session Accuracy: " + strconv.FormatFloat(MBSEStats.StatSum.AvgASession, 'f', 2, 64))
			core.NewText(frSStatsMain).SetText("Average Length (min) of a Session: " + strconv.FormatFloat(MBSEStats.StatSum.AvgMinSession, 'f', 2, 64))
			core.NewText(frSStatsMain).SetText("Average Session Score: " + strconv.FormatFloat(MBSEStats.StatSum.AvgScSession, 'f', 2, 64))
			btnSessionStats := core.NewButton(frSStatsMain).SetText("View more Session Statistics")
			btnSessionStats.OnClick(func(e events.Event) {
				diaS := core.NewBody("Sessions")
				tblSStats := core.NewTable(diaS).SetSlice(SessStats).BindSelect(&SSel)

				tblSStats.OnSelect(func(e events.Event) {
					diaSDetails := core.NewBody("Session Detailed Stats")
					frSDAll := core.NewFrame(diaSDetails)
					frSDAll.Styler(func(s *styles.Style) {
						s.Direction = styles.Column
					})
					frSDnotQ := core.NewFrame(frSDAll)
					frSDnotQ.Styler(func(s *styles.Style) {
						s.Direction = styles.Row
					})
					core.NewText(frSDnotQ).SetText("Player: " + SessStats[SSel].PDName)
					core.NewText(frSDnotQ).SetText("Session Length(min): " + strconv.Itoa(MBSEStats.SStats[SSel].TimePlayed))
					core.NewText(frSDnotQ).SetText("Number of Questions Answered: " + strconv.Itoa(len(MBSEStats.SStats[SSel].QStats)))
					core.NewText(frSDnotQ).SetText("Accuracy: " + strconv.FormatFloat(MBSEStats.SStats[SSel].Accuracy, 'f', 2, 64))
				
					core.NewTable(frSDAll).SetSlice(MBSEStats.SStats[SSel].QStats)
				
					diaSDetails.AddBottomBar(func(bar *core.Frame) {
						diaSDetails.AddOKOnly()
					})
				
					diaSDetails.RunFullDialog(tblSStats)
				})
			
				diaS.AddBottomBar(func(bar *core.Frame) {
					diaS.AddOKOnly()
				})
			
				diaS.RunFullDialog(btnSessionStats)
			})
		
			frQStatsMain := core.NewFrame(frStatSum)
			frQStatsMain.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
			})
			core.NewText(frQStatsMain).SetText("Overall Answering Accuracy")
			core.NewMeter(frQStatsMain).SetType(core.MeterCircle).SetMin(0).SetMax(100).SetValue(float32(MBSEStats.StatSum.AvgASession)).SetText(strconv.FormatFloat(MBSEStats.StatSum.AvgASession, 'f', 2, 64) + "%")
			btnQuestionStats := core.NewButton(frQStatsMain).SetText("View more Question Statistics")
			btnQuestionStats.OnClick(func(e events.Event) {
				diaQ := core.NewBody("Questions")
				core.NewTable(diaQ).SetSlice(QuesStats)
			
				diaQ.AddBottomBar(func(bar *core.Frame) {
					diaQ.AddOKOnly()
				})
			
				diaQ.RunFullDialog(btnQuestionStats)
			})
		
			frPStatsMain := core.NewFrame(frStatSum)
			frPStatsMain.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
			})
			core.NewText(frPStatsMain).SetText("Total number of players: " + strconv.Itoa(len(PlayStats)))
			core.NewMeter(frPStatsMain).SetType(core.MeterCircle).SetMin(0).SetMax(100).SetValue(float32(MBSEStats.StatSum.AvgAPlayers)).SetText("Average Improvement: " + strconv.FormatFloat(MBSEStats.StatSum.AvgAPlayers, 'f', 2, 64) + "%")
			btnPlayerStats := core.NewButton(frPStatsMain).SetText("View more Player Statistics")
			btnPlayerStats.OnClick(func(e events.Event) {
				diaP := core.NewBody("Players")
				core.NewTable(diaP).SetSlice(PlayStats)
			
				diaP.AddBottomBar(func(bar *core.Frame) {
					diaP.AddOKOnly()
				})
			
				diaP.RunFullDialog(btnPlayerStats)
			})

			winStats.AddBottomBar(func(bar *core.Frame) {
				winStats.AddOKOnly()
			})
			winStats.RunFullDialog(btnStatistics)
		})

		btnAdminPlay := core.NewButton(frMidAdmin).SetText("Play").SetIcon(icons.PlayCircleFill)
		btnAdminPlay.Styler(func(s *styles.Style) {
			s.Min.Set(units.Vw(20), units.Vh(20))
			s.Max.Set(units.Vw(20), units.Vh(20))
		})
		btnAdminPlay.OnClick(func(e events.Event) {
			pagesWidget.Open("Player")
		})

		//LowAdminFrame
		frLowAdmin := core.NewFrame(frAdminHome)
		frLowAdmin.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
			s.CenterAll()
			s.Gap.Set(units.Pw(40))
		})

		//Admin ManUsers Page
		btnUsers := core.NewButton(frLowAdmin).SetText("Users").SetIcon(icons.ManageAccountsFill)
		btnUsers.Styler(func(s *styles.Style) {
			s.Min.Set(units.Vw(20), units.Vh(20))
			s.Max.Set(units.Vw(20), units.Vh(20))
		})
		btnUsers.OnClick(func(e events.Event) {
			winUsers := core.NewBody("Manage Users")

			var intSelUIndex int
			tblUsers := core.NewTable(winUsers).SetSlice(&tableUsers).BindSelect(&intSelUIndex)
			tbUsers := core.NewToolbar(winUsers)
			tbUsers.Maker(func(p *tree.Plan) {
				tree.Add(p, func(w *core.Button) {
					w.SetIcon(icons.AddFill)
					w.OnClick(func(e events.Event) {
						diaAddUser := core.NewBody("Add User")
						diaAddUser.Styler(func(s *styles.Style) {
							s.Direction.SetString("Column")
						})
						//Message
						core.NewText(diaAddUser).SetText("Email:")
						txtEmail := core.NewTextField(diaAddUser)

						core.NewText(diaAddUser).SetText("Password:")
						txtPassword := core.NewTextField(diaAddUser)

						core.NewText(diaAddUser).SetText("First Name:")
						txtFName := core.NewTextField(diaAddUser)

						core.NewText(diaAddUser).SetText("Surname:")
						txtSName := core.NewTextField(diaAddUser)

						core.NewText(diaAddUser).SetText("Celnumber:")
						txtCNum := core.NewTextField(diaAddUser)

						core.NewText(diaAddUser).SetText("Display Name (optional):")
						txtDName := core.NewTextField(diaAddUser)

						//IsPosChange
						core.NewText(diaAddUser).SetText("Is an admin:")
						swAdmin := core.NewSwitch(diaAddUser)

						diaAddUser.AddBottomBar(func(bar *core.Frame) {
							diaAddUser.AddCancel(bar).OnClick(func(e events.Event) {
								core.MessageSnackbar(w, "User Not Added")
							})
							diaAddUser.AddOK(bar).OnClick(func(e events.Event) {
								MBSEGames.SignUp(ctx, firebaseapp, firestoreClient, txtEmail.Text(), txtPassword.Text(), txtFName.Text(), txtSName.Text(), txtCNum.Text(), txtDName.Text(), swAdmin.IsChecked())
								core.MessageSnackbar(w, "User Added Successfully!")
							})
						})
						diaAddUser.RunDialog(w)
					})
				})

				tree.Add(p, func(w *core.Button) {
					w.SetIcon(icons.EditFill)
					w.OnClick(func(e events.Event) {
						diaEditUser := core.NewBody("Edit User")
						diaEditUser.Styler(func(s *styles.Style) {
							s.Direction.SetString("Column")
						})
						//Message
						core.NewText(diaEditUser).SetText("Display Name:")
						txtDName := core.NewTextField(diaEditUser).SetPlaceholder(Users[tblUsers.SelectedIndex].DisplayName)

						//IsPosChange
						core.NewText(diaEditUser).SetText("First Name:")
						txtFName := core.NewTextField(diaEditUser).SetPlaceholder(Users[tblUsers.SelectedIndex].FirstName)

						//IsScoreChange
						core.NewText(diaEditUser).SetText("Surname:")
						txtSName := core.NewTextField(diaEditUser).SetPlaceholder(Users[tblUsers.SelectedIndex].Surname)

						//IsLMove
						core.NewText(diaEditUser).SetText("Celnumber:")
						txtCel := core.NewTextField(diaEditUser).SetPlaceholder(Users[tblUsers.SelectedIndex].CelNum)

						diaEditUser.AddBottomBar(func(bar *core.Frame) {
							diaEditUser.AddCancel(bar).OnClick(func(e events.Event) {
								core.MessageSnackbar(w, "User Not Edited")
							})
							diaEditUser.AddOK(bar).OnClick(func(e events.Event) {
								if txtDName.Text() != Users[tblUsers.SelectedIndex].DisplayName {
									MBSEGames.EditUser(ctx, firestoreClient, Users[tblUsers.SelectedIndex].DocRef, "DisplayName", txtDName.Text(), 0, 0)
								}

								if txtFName.Text() != Users[tblUsers.SelectedIndex].FirstName {
									MBSEGames.EditUser(ctx, firestoreClient, Users[tblUsers.SelectedIndex].DocRef, "FirstName", txtFName.Text(), 0, 0)
								}

								if txtSName.Text() != Users[tblUsers.SelectedIndex].Surname {
									MBSEGames.EditUser(ctx, firestoreClient, Users[tblUsers.SelectedIndex].DocRef, "Surname", txtSName.Text(), 0, 0)
								}

								if txtCel.Text() != Users[tblUsers.SelectedIndex].CelNum {
									MBSEGames.EditUser(ctx, firestoreClient, Users[tblUsers.SelectedIndex].DocRef, "CelNum", txtCel.Text(), 0, 0)
								}

								core.MessageSnackbar(w, "User Edited Successfully!")
							})
						})
						diaEditUser.RunDialog(w)
					})
				})
			})
			winUsers.RunFullDialog(btnUsers)
		})

		//Admin Sessions Page
		btnSessions := core.NewButton(frLowAdmin).SetText("Sessions").SetIcon(icons.DatasetFill)
		btnSessions.Styler(func(s *styles.Style) {
			s.Min.Set(units.Vw(20), units.Vh(20))
			s.Max.Set(units.Vw(20), units.Vh(20))
		})
		btnSessions.OnClick(func(e events.Event) {
			winSessions := core.NewBody("View Sessions")
			frSessions := core.NewFrame(winSessions)
			frSessions.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
			})
			core.NewTable(winSessions).SetSlice(&Sessions).SetReadOnly(true)
			btnToCSV := core.NewButton(winSessions).SetIcon(icons.ExportNotesFill)
			btnToCSV.OnClick(func(e events.Event) {
				file, err := os.Create("Sessions.csv")
				if err != nil {
					log.Fatalln("Failed to create file:", err)
				}
				defer file.Close()

				writer := csv.NewWriter(file)
				defer writer.Flush()

				var data [][]string
				headers := []string{
					"Start Time", "End Time", "Score", "DisplayName", "Gamemode", "Ai1", "Ai2", "Ai3", "Total Correct", "Total Incorrect",
				}
				data = append(data, headers)
				for _, ses := range MBSEGames.Sessions {
					var sesdata []string 
					sesdata = append(sesdata, ses.StartTime)
					sesdata = append(sesdata, ses.EndTime)
					sesdata = append(sesdata, strconv.Itoa(int(ses.Score)))
					sesdata = append(sesdata, ses.UserDisplayName)
					sesdata = append(sesdata, ses.Gamemode)
					sesdata = append(sesdata, ses.Ai1)
					sesdata = append(sesdata, ses.Ai2)
					sesdata = append(sesdata, ses.Ai3)
					sesdata = append(sesdata, strconv.Itoa(ses.NumCorrect))
					sesdata = append(sesdata, strconv.Itoa(ses.NumIncorrect))

					data = append(data, sesdata)
				}

				if err := writer.WriteAll(data); err != nil {
					log.Fatalln("Error Exporting to CSV:", err)
				} 
			})
			winSessions.RunFullDialog(btnSessions)
		})
	})

	/////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	//////Player Home Page
	/////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	pagesWidget.AddPage("Player", func(pg *core.Pages) {
		img, _, err := imagex.OpenFS(embeddedFiles, "Assets/Images/BackgroundV3.png")
		imgBackground := imagex.Resize(img, image.Point{2000, 2000})
		if err != nil {
			log.Fatalf("Error loading image: %s", err)
		}

		pg.Frame.Styler(func(s *styles.Style) {
			s.Background = imgBackground
		})
		playerTabs := core.NewTabs(pg)
		frpHome, btnpHome := playerTabs.NewTab("Home")
		btnpHome.SetIcon(icons.Home)
		frpHome.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.CenterAll()
		})
		core.NewText(frpHome).SetText("Welcome " + CUserName + "!")
		icPfp := core.NewIconButton(frpHome)
		switch CurrentUser.PicID {
		case "Payments":
			icPfp.SetIcon(icons.PaymentsFill)
		case "Sell":
			icPfp.SetIcon(icons.SellFill)
		case "Database":
			icPfp.SetIcon(icons.DatabaseFill)
		case "TrendingUp":
			icPfp.SetIcon(icons.TrendingUpFill)
		case "Savings":
			icPfp.SetIcon(icons.SavingsFill)
		case "CreditCard":
			icPfp.SetIcon(icons.CreditCardFill)
		case "ReceiptLong":
			icPfp.SetIcon(icons.ReceiptLongFill)
		case "CardMembership":
			icPfp.SetIcon(icons.CardMembershipFill)
		default:
			icPfp.SetIcon(icons.PaymentsFill)
		}
		icPfp.Styler(func(s *styles.Style) {
			s.IconSize.Set(units.Vh(25))
			s.Color = colors.Uniform(colors.FromRGB(0, 102, 255))
		})
		icPfp.OnClick(func(e events.Event) {
			dUserInfo := core.NewBody("Settings")
			frUserInfo := core.NewFrame(dUserInfo)
			frUserInfo.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
				s.CenterAll()
			})
			swNewUserName := core.NewSwitch(frUserInfo).SetType(core.SwitchCheckbox).SetText("Generate New UserName")
			chUserIcon := core.NewChooser(frUserInfo).SetPlaceholder("Select an icon").SetItems(
				core.ChooserItem{Value: "Payments", Icon: icons.PaymentsFill},
				core.ChooserItem{Value: "Sell", Icon: icons.SellFill},
				core.ChooserItem{Value: "TrendingUp", Icon: icons.TrendingUpFill},
				core.ChooserItem{Value: "Savings", Icon: icons.SavingsFill},
				core.ChooserItem{Value: "CreditCard", Icon: icons.CreditCardFill},
				core.ChooserItem{Value: "ReceiptLong", Icon: icons.ReceiptLongFill},
				core.ChooserItem{Value: "CardMembership", Icon: icons.CardMembershipFill},
			)
			dUserInfo.AddBottomBar(func(bar *core.Frame) {
				dUserInfo.AddCancel(bar)
				dUserInfo.AddOK(bar).OnClick(func(e events.Event) {
					if swNewUserName.IsChecked() {
						err, strNewUN := MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.UID, "DisplayName", "generate", 0, 0)

						if err != nil {
							log.Fatalf("Error updating username: %s", err)
						}

						CurrentUser.DisplayName = strNewUN
					}
					if chUserIcon.CurrentItem.Value != nil {
						err, _ := MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.UID, "PicID", chUserIcon.CurrentItem.GetText(), 0, 0)

						if err != nil {
							log.Fatalf("Error updating icon: %s", err)
						}

						CurrentUser.PicID = chUserIcon.CurrentItem.GetText()
						switch CurrentUser.PicID {
						case "Payments":
							icPfp.SetIcon(icons.PaymentsFill)
						case "Sell":
							icPfp.SetIcon(icons.SellFill)
						case "Database":
							icPfp.SetIcon(icons.DatabaseFill)
						case "TrendingUp":
							icPfp.SetIcon(icons.TrendingUpFill)
						case "Savings":
							icPfp.SetIcon(icons.SavingsFill)
						case "CreditCard":
							icPfp.SetIcon(icons.CreditCardFill)
						case "ReceiptLong":
							icPfp.SetIcon(icons.ReceiptLongFill)
						case "CardMembership":
							icPfp.SetIcon(icons.CardMembershipFill)
						default:
							icPfp.SetIcon(icons.PaymentsFill)
						}
					}
				})
			})

			dUserInfo.RunDialog(icPfp)
		})
		core.NewText(frpHome).SetText("Current Streak " + strconv.Itoa(CurrentUser.Streak) + "!")
		core.NewText(frpHome).SetText("Total Score " + strconv.Itoa(int(CurrentUser.TotalScore)) + "!")

		nanSec := CurrentUser.TotalTime
		sec := nanSec / 1000000000
		min := sec / 60
		hour := min / 60
		days := hour / 24
		strTime := "You've played for a total of " + strconv.FormatInt(days, 10) + " day/s " + strconv.FormatInt(hour, 10) + " hour/s " + strconv.FormatInt(min, 10) + " minute/s and " + strconv.FormatInt(sec, 10) + " second/s!"
		core.NewText(frpHome).SetText(strTime)

		//Leaderboards
		frpLead, btnpLead := playerTabs.NewTab("Leaderboards")
		btnpLead.SetIcon(icons.SocialLeaderboard)
		frpLead.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
			s.Gap.X.Set(15, units.UnitVw)
			s.Min.Set(units.Vw(100), units.Vh(95))
			s.Grow.Set(1, 1)
			s.Background = colors.Uniform(color.Transparent)
			s.CenterAll()
		})
		frleadStreak := core.NewFrame(frpLead)
		frleadStreak.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Min.Set(units.Vw(20), units.Vh(95))
			s.Grow.Set(0, 1)
			s.CenterAll()
		})
		core.NewText(frleadStreak).SetText("Streak Leaders")
		core.NewTable(frleadStreak).SetSlice(&disStL).SetReadOnly(true)

		frleadScore := core.NewFrame(frpLead)
		frleadScore.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Min.Set(units.Vw(20), units.Vh(95))
			s.Grow.Set(0, 1)
			s.CenterAll()
		})
		core.NewText(frleadScore).SetText("Score Leaders")
		core.NewTable(frleadScore).SetSlice(&disScL).SetReadOnly(true)

		frleadTime := core.NewFrame(frpLead)
		frleadTime.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Min.Set(units.Vw(20), units.Vh(95))
			s.Grow.Set(0, 1)
			s.CenterAll()
		})
		core.NewText(frleadTime).SetText("Time Leaders")
		core.NewTable(frleadTime).SetSlice(&disTL).SetReadOnly(true)

		//Setup Page
		frPlay, btnPlay := playerTabs.NewTab("Play")
		btnPlay.SetIcon(icons.PlayCircle)
		frPlay.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.CenterAll()
		})
		core.NewText(frPlay).SetText("Press the button below to get started!")

		btnGamePlay := core.NewButton(frPlay).SetText("Play").SetIcon(icons.PlayCircle)
		btnGamePlay.OnClick(func(e events.Event) {
			diSetup := core.NewBody("Game Setup")
			core.NewText(diSetup).SetType(core.TextSupporting).SetText("Select all options")
			chGameMode := core.NewChooser(diSetup).SetPlaceholder("Select game mode").SetStrings("Easy", "Medium", "Hard")
			chAi1 := core.NewChooser(diSetup).SetPlaceholder("Select Ai").SetStrings("Numinor", "Samuel", "Ajani", "Trell", "Quintin", "Quasar")
			chAi2 := core.NewChooser(diSetup).SetPlaceholder("Select Ai").SetStrings("Numinor", "Samuel", "Ajani", "Trell", "Quintin", "Quasar")
			chAi3 := core.NewChooser(diSetup).SetPlaceholder("Select Ai").SetStrings("Numinor", "Samuel", "Ajani", "Trell", "Quintin", "Quasar")
			diSetup.AddBottomBar(func(bar *core.Frame) {
				diSetup.AddCancel(bar).OnClick(func(e events.Event) {
					core.MessageSnackbar(btnGamePlay, "Setup canceled")
				})
				diSetup.AddOK(bar).OnClick(func(e events.Event) {
					GameMode = chGameMode.CurrentItem.GetText()

					if chAi1.CurrentItem.GetText() == chAi2.CurrentItem.GetText() || chAi2.CurrentItem.GetText() == chAi3.CurrentItem.GetText() || chAi3.CurrentItem.GetText() == chAi1.CurrentItem.GetText() {
						core.MessageSnackbar(btnGamePlay, "Please select an Ai only once")
					} else if chAi1.CurrentItem.GetText() == "" || chAi1.CurrentItem.GetText() == "Select Ai" || chAi2.CurrentItem.GetText() == "" || chAi2.CurrentItem.GetText() == "Select Ai" || chAi3.CurrentItem.GetText() == "" || chAi3.CurrentItem.GetText() == "Select Ai" || chGameMode.CurrentItem.GetText() == "" || chGameMode.CurrentItem.GetText() == "Select game mode" {
						core.MessageSnackbar(btnGamePlay, "Please select an option in each list")
					} else if chGameMode.CurrentItem.GetText() == "Hard" && CurrentUser.TotalScore < 10000 {
						core.MessageSnackbar(btnGamePlay, "Your Score isn't high enough for that mode yet!")
					} else if chGameMode.CurrentItem.GetText() == "Medium" && CurrentUser.TotalScore < 4000 {
						core.MessageSnackbar(btnGamePlay, "Your Score isn't high enough for that mode yet!")
					} else {
						Session.Ai1 = chAi1.CurrentItem.GetText()
						Session.Ai2 = chAi2.CurrentItem.GetText()
						Session.Ai3 = chAi3.CurrentItem.GetText()
						Session.Gamemode = chGameMode.CurrentItem.GetText()
						Session.Subject = "FAIS"
						switch chAi1.CurrentItem.GetText() {
						case "Numinor":
							switch GameMode {
							case "Easy":
								Ai1 = 0
							case "Medium":
								Ai1 = 1
							case "Hard":
								Ai1 = 2
							}

						case "Samuel":
							switch GameMode {
							case "Easy":
								Ai1 = 3
							case "Medium":
								Ai1 = 4
							case "Hard":
								Ai1 = 5
							}

						case "Ajani":
							switch GameMode {
							case "Easy":
								Ai1 = 6
							case "Medium":
								Ai1 = 7
							case "Hard":
								Ai1 = 8
							}

						case "Trell":
							switch GameMode {
							case "Easy":
								Ai1 = 9
							case "Medium":
								Ai1 = 10
							case "Hard":
								Ai1 = 11
							}

						case "Quintin":
							switch GameMode {
							case "Easy":
								Ai1 = 12
							case "Medium":
								Ai1 = 13
							case "Hard":
								Ai1 = 14
							}

						case "Quasar":
							switch GameMode {
							case "Easy":
								Ai1 = 15
							case "Medium":
								Ai1 = 16
							case "Hard":
								Ai1 = 17
							}
						}

						switch chAi2.CurrentItem.GetText() {
						case "Numinor":
							switch GameMode {
							case "Easy":
								Ai2 = 0
							case "Medium":
								Ai2 = 1
							case "Hard":
								Ai2 = 2
							}

						case "Samuel":
							switch GameMode {
							case "Easy":
								Ai2 = 3
							case "Medium":
								Ai2 = 4
							case "Hard":
								Ai2 = 5
							}

						case "Ajani":
							switch GameMode {
							case "Easy":
								Ai2 = 6
							case "Medium":
								Ai2 = 7
							case "Hard":
								Ai2 = 8
							}

						case "Trell":
							switch GameMode {
							case "Easy":
								Ai2 = 9
							case "Medium":
								Ai2 = 10
							case "Hard":
								Ai2 = 11
							}

						case "Quintin":
							switch GameMode {
							case "Easy":
								Ai2 = 12
							case "Medium":
								Ai2 = 13
							case "Hard":
								Ai2 = 14
							}

						case "Quasar":
							switch GameMode {
							case "Easy":
								Ai2 = 15
							case "Medium":
								Ai2 = 16
							case "Hard":
								Ai2 = 17
							}
						}

						switch chAi3.CurrentItem.GetText() {
						case "Numinor":
							switch GameMode {
							case "Easy":
								Ai3 = 0
							case "Medium":
								Ai3 = 1
							case "Hard":
								Ai3 = 2
							}

						case "Samuel":
							switch GameMode {
							case "Easy":
								Ai3 = 3
							case "Medium":
								Ai3 = 4
							case "Hard":
								Ai3 = 5
							}

						case "Ajani":
							switch GameMode {
							case "Easy":
								Ai3 = 6
							case "Medium":
								Ai3 = 7
							case "Hard":
								Ai3 = 8
							}

						case "Trell":
							switch GameMode {
							case "Easy":
								Ai3 = 9
							case "Medium":
								Ai3 = 10
							case "Hard":
								Ai3 = 11
							}

						case "Quintin":
							switch GameMode {
							case "Easy":
								Ai3 = 12
							case "Medium":
								Ai3 = 13
							case "Hard":
								Ai3 = 14
							}

						case "Quasar":
							switch GameMode {
							case "Easy":
								Ai3 = 15
							case "Medium":
								Ai3 = 16
							case "Hard":
								Ai3 = 17
							}
						}
						for _, q := range Questions {
							switch Session.Gamemode {
							case "Easy":
								if q.Level == 1 {
									switch q.Category {
									case "CCC": ccc = append(ccc, q)
									case "MMM": mmm = append(mmm, q)
									case "LBL": lbl = append(lbl, q)
									case "TTT": ttt = append(ttt, q)
									case "Debarred": de = append(de, q)
									case "Joke": lol = append(lol, q)
									case "Complaints": co = append(co, q)
									}
									lQs = append(lQs, q)
								}
							case "Medium":
								if q.Level == 2 {
									switch q.Category {
									case "CCC": ccc = append(ccc, q)
									case "MMM": mmm = append(mmm, q)
									case "LBL": lbl = append(lbl, q)
									case "TTT": ttt = append(ttt, q)
									case "Debarred": de = append(de, q)
									case "Joke": lol = append(lol, q)
									case "Complaints": co = append(co, q)
									}
									lQs = append(lQs, q)
								}
							case "Hard":
								if q.Level == 3 {
									switch q.Category {
									case "CCC": ccc = append(ccc, q)
									case "MMM": mmm = append(mmm, q)
									case "LBL": lbl = append(lbl, q)
									case "TTT": ttt = append(ttt, q)
									case "Debarred": de = append(de, q)
									case "Joke": lol = append(lol, q)
									case "Complaints": co = append(co, q)
									}
									lQs = append(lQs, q)
								}
							}
						}
						pagesWidget.Open("Moonopoly")
						startTime = time.Now()
					}
				})
			})
			diSetup.RunDialog(btnGamePlay)
		})
	})

	/////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	//////Game Page
	/////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	pagesWidget.AddPage("Moonopoly", func(pg *core.Pages) {
		//load board & setup specs
		var winSize, boardSize image.Point
		winSize = system.TheApp.Window(0).Size()
		boardSize.X = winSize.Y
		boardSize.Y = winSize.Y
		img, _, err := imagex.OpenFS(embeddedFiles, "Assets/Images/Moonopoly.png")
		imgBoard := imagex.Resize(img, boardSize)
		if err != nil {
			log.Fatalf("Error loading image: %s", err)
		}

		pg.Frame.Styler(func(s *styles.Style) {
			s.Background = imgBoard
			s.Display = styles.Custom
			s.ObjectFit = styles.FitScaleDown
			s.Min.Set(units.Vw(100), units.Vh(100)) //1 Tile(1/6) of 850Dp is 141,666666666666666666666666666666666666666666Dp
			s.Overflow.X.SetString("OverflowScroll")
			s.Overflow.Y.SetString("OverflowScroll")
			s.Grow.Set(1, 1)
		})

		//Scores Display
		var intai1Score, intai2Score, intai3Score int
		playerScore = 0
		intai1Score = 0
		intai2Score = 0
		intai3Score = 0

		//Player Score display
		frplayerScore := core.NewFrame(&pg.Frame)
		frplayerScore.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
			s.Min.X.Set(20, units.UnitEm)
			s.Pos.Set(units.Vw(60), units.Vh(5))
			s.RenderBox = false
			s.Background = colors.Uniform(colors.FromRGB(18, 19, 22))
		})
		playerTokenID := core.NewIcon(frplayerScore)
		playerTokenID.Styler(func(s *styles.Style) {
			s.IconSize.Set(units.Vh(4.1666666667))
			s.Color = colors.Uniform(colors.FromRGB(0, 102, 255))
		})
		txtplayerScore := core.Bind(&strPlayerScoreLbl, core.NewText(frplayerScore))
		strPlayerScoreLbl = CUserName + ": R" + strconv.Itoa(playerScore)
		txtplayerScore.Update()

		frai1Score := core.NewFrame(&pg.Frame)
		frai1Score.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
			s.Min.X.Set(20, units.UnitEm)
			s.Pos.Set(units.Vw(60), units.Vh(30))
			s.RenderBox = false
			s.Background = colors.Uniform(colors.FromRGB(18, 19, 22))
		})
		ai1TokenID := core.NewIcon(frai1Score)
		ai1TokenID.Styler(func(s *styles.Style) {
			s.IconSize.Set(units.Vh(4.1666666667))
			s.Color = colors.Uniform(colors.FromRGB(255, 47, 0))
		})
		txtai1Score := core.NewText(frai1Score)
		txtai1Score.SetText(ais[Ai1].Name + ": R" + strconv.Itoa(intai1Score))

		frai2Score := core.NewFrame(&pg.Frame)
		frai2Score.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
			s.Min.X.Set(20, units.UnitEm)
			s.Pos.Set(units.Vw(60), units.Vh(55))
			s.RenderBox = false
			s.Background = colors.Uniform(colors.FromRGB(18, 19, 22))
		})
		ai2TokenID := core.NewIcon(frai2Score)
		ai2TokenID.Styler(func(s *styles.Style) {
			s.IconSize.Set(units.Vh(4.1666666667))
			s.Color = colors.Uniform(colors.FromRGB(255, 47, 0))
		})
		txtai2Score := core.NewText(frai2Score)
		txtai2Score.SetText(ais[Ai2].Name + ": R" + strconv.Itoa(intai2Score))

		frai3Score := core.NewFrame(&pg.Frame)
		frai3Score.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
			s.Min.X.Set(20, units.UnitEm)
			s.Pos.Set(units.Vw(60), units.Vh(80))
			s.RenderBox = false
			s.Background = colors.Uniform(colors.FromRGB(18, 19, 22))
		})
		ai3TokenID := core.NewIcon(frai3Score)
		ai3TokenID.Styler(func(s *styles.Style) {
			s.IconSize.Set(units.Vh(4.1666666667))
			s.Color = colors.Uniform(colors.FromRGB(255, 47, 0))
		})
		txtai3Score := core.NewText(frai3Score)
		txtai3Score.SetText(ais[Ai3].Name + ": R" + strconv.Itoa(intai3Score))

		switch CurrentUser.PicID {
		case "Payments":
			playerTokenID.SetIcon(icons.PaymentsFill)
		case "Sell":
			playerTokenID.SetIcon(icons.SellFill)
		case "Database":
			playerTokenID.SetIcon(icons.DatabaseFill)
		case "TrendingUp":
			playerTokenID.SetIcon(icons.TrendingUpFill)
		case "Savings":
			playerTokenID.SetIcon(icons.SavingsFill)
		case "CreditCard":
			playerTokenID.SetIcon(icons.CreditCardFill)
		case "ReceiptLong":
			playerTokenID.SetIcon(icons.ReceiptLongFill)
		case "CardMembership":
			playerTokenID.SetIcon(icons.CardMembershipFill)
		default:
			playerTokenID.SetIcon(icons.PaymentsFill)
		}

		switch Session.Ai1 {
		case "Numinor":
			ai1TokenID.SetIcon(icons.PlayingCardsFill)
		case "Samuel":
			ai1TokenID.SetIcon(icons.HourglassTopFill)
		case "Ajani":
			ai1TokenID.SetIcon(icons.BuildFill)
		case "Trell":
			ai1TokenID.SetIcon(icons.SettingsFill)
		case "Quintin":
			ai1TokenID.SetIcon(icons.ReceiptFill)
		case "Quasar":
			ai1TokenID.SetIcon(icons.StarFill)
		}

		switch Session.Ai2 {
		case "Numinor":
			ai2TokenID.SetIcon(icons.PlayingCardsFill)
		case "Samuel":
			ai2TokenID.SetIcon(icons.HourglassTopFill)
		case "Ajani":
			ai2TokenID.SetIcon(icons.BuildFill)
		case "Trell":
			ai2TokenID.SetIcon(icons.SettingsFill)
		case "Quintin":
			ai2TokenID.SetIcon(icons.ReceiptFill)
		case "Quasar":
			ai2TokenID.SetIcon(icons.StarFill)
		}

		switch Session.Ai3 {
		case "Numinor":
			ai3TokenID.SetIcon(icons.PlayingCardsFill)
		case "Samuel":
			ai3TokenID.SetIcon(icons.HourglassTopFill)
		case "Ajani":
			ai3TokenID.SetIcon(icons.BuildFill)
		case "Trell":
			ai3TokenID.SetIcon(icons.SettingsFill)
		case "Quintin":
			ai3TokenID.SetIcon(icons.ReceiptFill)
		case "Quasar":
			ai3TokenID.SetIcon(icons.StarFill)
		}

		btnExitGame := core.NewButton(&pg.Frame).SetIcon(icons.KeyboardReturnFill)
		btnExitGame.Styler(func(s *styles.Style) {
			s.Pos.Set(units.Vw(75), units.Vh(55))
			s.Background = colors.Uniform(colors.FromRGB(24, 4, 48))
			s.Color = colors.Uniform(colors.FromRGB(134, 0, 204))
		})

		btnExitGame.OnClick(func(e events.Event) {
			timeNow := time.Now()
			numNanosec := timeNow.Sub(startTime).Nanoseconds()
			CurrentUser.TotalTime += numNanosec
			CurrentUser.TotalScore += Session.Score
			err = MBSEGames.AddSession(ctx, firestoreClient, CurrentUser.DisplayName, Session.Subject, Session.Gamemode, Session.Ai1, Session.Ai2, Session.Ai3, Session.StartTime, timeNow.Format(time.RFC822), Session.QDetails, Session.Score, Session.NumCorrect, Session.NumIncorrect)
			if err != nil {
				log.Fatalf("error adding session: %s", err)
			}

			_, nMonth, nDay := timeNow.Date()
			_, lMonth, lDay := CurrentUser.LastPlayed.Date()
			timeBetween := timeNow.Sub(CurrentUser.LastPlayed).Hours()
			bStreakAdd := false
			if nMonth != lMonth && nMonth - lMonth == 1 && nDay == 1 {	
				switch nMonth {
				case 1:
					if lDay == 31 {
						bStreakAdd = true
					}
				case 2:
					if lDay == 28 {
						bStreakAdd = true
					}
				case 3:
					if lDay == 31 {
						bStreakAdd = true
					}
				case 4:
					if lDay == 30 {
						bStreakAdd = true
					}
				case 5:
					if lDay == 31 {
						bStreakAdd = true
					}
				case 6:
					if lDay == 30 {
						bStreakAdd = true
					}
				case 7:
					if lDay == 31 {
						bStreakAdd = true
					}
				case 8:
					if lDay == 31 {
						bStreakAdd = true
					}
				case 9:
					if lDay == 30 {
						bStreakAdd = true
					}
				case 10:
					if lDay == 31 {
						bStreakAdd = true
					}
				case 11:
					if lDay == 30 {
						bStreakAdd = true
					}
				case 12:
					if lDay == 31 {
						bStreakAdd = true
					} 
				}
			} else if nDay - lDay == 1 && nMonth == lMonth {
				bStreakAdd = true
			}
			if bStreakAdd {
				CurrentUser.Streak++
				err, _ = MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.DocRef, "Streak", "", CurrentUser.Streak, 0)
				if err != nil {
					log.Fatalf("error updating streak: %s", err)
				}
				if CurrentUser.Streak > CurrentUser.RecordStreak {
					err, _ = MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.DocRef, "RecordStreak", "", CurrentUser.Streak, 0)
					if err != nil {
						log.Fatalf("error updating record streak: %s", err)
					}
				}
			} else if timeBetween > 48 {
				CurrentUser.Streak = 1
				err, _ = MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.DocRef, "Streak", "", CurrentUser.Streak, 0)
				if err != nil {
					log.Fatalf("error updating streak: %s", err)
				}
			}

			err, _ = MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.DocRef, "LastDate", timeNow.Format(time.RFC3339), 0, 0)
			if err != nil {
				log.Fatalf("error updating last date: %s", err)
			}

			err, _ = MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.DocRef, "TotalTime", "", 0, CurrentUser.TotalTime)
			if err != nil {
				log.Fatalf("error updating total time: %s", err)
			}

			err, _ = MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.DocRef, "TotalScore", "", 0, CurrentUser.TotalScore)
			if err != nil {
				log.Fatalf("error updating total score: %s", err)
			}
			pagesWidget.Open("Player")
		})

		//Game board
		var XVal, YVal units.Value
		XVal.Set(0, units.UnitVh)
		YVal.Set(0, units.UnitVh)

		var ai1Miss, ai2Miss, ai3Miss bool
		playerMiss = false
		ai1Miss = false
		ai2Miss = false
		ai3Miss = false

		playerToken := core.NewIcon(&pg.Frame).SetIcon(icons.PaidFill)
		playerToken.Styler(func(s *styles.Style) {
			s.Pos.Set(units.Vh(2.5), units.Vh(87.5))
			s.Min.Set(units.Vh(4.1666666667), units.Vh(4.1666666667))
			s.Max.Set(units.Vh(4.1666666667), units.Vh(4.1666666667))
			s.Grow.Set(1, 1)
			s.Color = colors.Uniform(colors.FromRGB(0, 102, 255))
		})
		ai1Token := core.NewIcon(&pg.Frame)
		ai1Token.Styler(func(s *styles.Style) {
			s.Pos.Set(units.Vh(10), units.Vh(87.5))
			s.Min.Set(units.Vh(4.1666666667), units.Vh(4.1666666667))
			s.Max.Set(units.Vh(4.1666666667), units.Vh(4.1666666667))
			s.Grow.Set(1, 1)
			s.Color = colors.Uniform(colors.FromRGB(255, 47, 0))
		})
		ai2Token := core.NewIcon(&pg.Frame)
		ai2Token.Styler(func(s *styles.Style) {
			s.Pos.Set(units.Vh(2.5), units.Vh(93.3333333333))
			s.Min.Set(units.Vh(4.1666666667), units.Vh(4.1666666667))
			s.Max.Set(units.Vh(4.1666666667), units.Vh(4.1666666667))
			s.Grow.Set(1, 1)
			s.Color = colors.Uniform(colors.FromRGB(255, 47, 0))
		})
		ai3Token := core.NewIcon(&pg.Frame)
		ai3Token.Styler(func(s *styles.Style) {
			s.Pos.Set(units.Vh(10), units.Vh(93.3333333333))
			s.Min.Set(units.Vh(4.1666666667), units.Vh(4.1666666667))
			s.Max.Set(units.Vh(4.1666666667), units.Vh(4.1666666667))
			s.Grow.Set(1, 1)
			s.Color = colors.Uniform(colors.FromRGB(255, 47, 0))
		})

		switch CurrentUser.PicID {
		case "Payments":
			playerToken.SetIcon(icons.PaymentsFill)
		case "Sell":
			playerToken.SetIcon(icons.SellFill)
		case "Database":
			playerToken.SetIcon(icons.DatabaseFill)
		case "TrendingUp":
			playerToken.SetIcon(icons.TrendingUpFill)
		case "Savings":
			playerToken.SetIcon(icons.SavingsFill)
		case "CreditCard":
			playerToken.SetIcon(icons.CreditCardFill)
		case "ReceiptLong":
			playerToken.SetIcon(icons.ReceiptLongFill)
		case "CardMembership":
			playerToken.SetIcon(icons.CardMembershipFill)
		default:
			playerToken.SetIcon(icons.PaymentsFill)
		}
		
		switch Session.Ai1 {
		case "Numinor":
			ai1Token.SetIcon(icons.PlayingCardsFill)
		case "Samuel":
			ai1Token.SetIcon(icons.HourglassTopFill)
		case "Ajani":
			ai1Token.SetIcon(icons.BuildFill)
		case "Trell":
			ai1Token.SetIcon(icons.SettingsFill)
		case "Quintin":
			ai1Token.SetIcon(icons.ReceiptFill)
		case "Quasar":
			ai1Token.SetIcon(icons.StarFill)
		}

		switch Session.Ai2 {
		case "Numinor":
			ai2Token.SetIcon(icons.PlayingCardsFill)
		case "Samuel":
			ai2Token.SetIcon(icons.HourglassTopFill)
		case "Ajani":
			ai2Token.SetIcon(icons.BuildFill)
		case "Trell":
			ai2Token.SetIcon(icons.SettingsFill)
		case "Quintin":
			ai2Token.SetIcon(icons.ReceiptFill)
		case "Quasar":
			ai2Token.SetIcon(icons.StarFill)
		}

		switch Session.Ai3 {
		case "Numinor":
			ai3Token.SetIcon(icons.PlayingCardsFill)
		case "Samuel":
			ai3Token.SetIcon(icons.HourglassTopFill)
		case "Ajani":
			ai3Token.SetIcon(icons.BuildFill)
		case "Trell":
			ai3Token.SetIcon(icons.SettingsFill)
		case "Quintin":
			ai3Token.SetIcon(icons.ReceiptFill)
		case "Quasar":
			ai3Token.SetIcon(icons.StarFill)
		}

		//var intplayerPos, intai1Pos, intai2Pos, intai3Pos int
		intplayerPos = 1
		intai1Pos = 1
		intai2Pos = 1
		intai3Pos = 1
		//gameplay logic
		btnRoll := core.NewButton(&pg.Frame).SetText("Roll")
		btnRoll.Styler(func(s *styles.Style) {
			s.Pos.Set(units.Vw(75), units.Vh(45))
			s.Background = colors.Uniform(colors.FromRGB(24, 4, 48))
			s.Color = colors.Uniform(colors.FromRGB(134, 0, 204))
		})
		btnRoll.OnClick(func(e events.Event) {
			if Session.StartTime == "" {
				Session.StartTime = time.Now().Format(time.RFC822)
			}

			r := *rand.New(rand.NewSource(time.Now().Unix()))
			go func() {
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//AI1
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
				if ai1Miss {
					randpercentChance := r.Intn(100) + 1
					if ais[Ai1].StatsSection5 >= randpercentChance {
						intai1Score += 100
						txtai1Score.SetText(ais[Ai1].Name + ": R" + strconv.Itoa(intai1Score)).Update()
						ai1Miss = false
					}
				} else {
					intRoll := r.Intn(6) + 1
					i := intRoll
					intScoreGain := Move(intai1Pos, i, 2, &pg.Frame, ai1Token)
					intai1Pos += intRoll
					if intScoreGain != 0 {
						intai1Score += intScoreGain
						intai1Pos -= 20
					}
					switch intai1Pos {
					case 1:
						randpercentChance := r.Intn(100) + 1
						if ais[Ai1].StatsSection5 >= randpercentChance {
							intai1Score += 100
							txtai1Score.SetText(ais[Ai1].Name + ": R" + strconv.Itoa(intai1Score)).Update()
						}
					case 2, 3, 5:
						randpercentChance := r.Intn(100) + 1
						if ais[Ai1].StatsSection1 >= randpercentChance {
							intai1Score += 100
							txtai1Score.SetText(ais[Ai1].Name + ": R" + strconv.Itoa(intai1Score)).Update()
							ai1Miss = false
						}
					case 4, 8, 14, 18:
						if len(ChanceCards) <= 0 {
							log.Fatalf("ChanceCards <= 0")
						}
						randCCard := r.Intn(len(ChanceCards))
						if ChanceCards[randCCard].IsScoreChange {

							intai1Score += ChanceCards[randCCard].Value
							txtai1Score.SetText(ais[Ai1].Name + ": R" + strconv.Itoa(intai1Score)).Update()

						} else if !ChanceCards[randCCard].IsToFieldMove {

							if ChanceCards[randCCard].IsPosChange {
								i = ChanceCards[randCCard].Value
								
								intScoreGain = Move(intai1Pos, i, 2, &pg.Frame, ai1Token)
								intai1Pos += i
								if intScoreGain != 0 {
									intai1Score += intScoreGain
									intai1Pos -= 20
								}
							} else {
								i = ChanceCards[randCCard].Value
								RevMove(intai1Pos, i, 2, &pg.Frame, ai1Token)
								intai1Pos -= i
								if intai1Pos < 1 {
									intai1Pos += 20
								}
							}
							randpercentChance := r.Intn(100) + 1
							if ais[Ai1].StatsSection5 >= randpercentChance {
								intai1Score += 100
								txtai1Score.SetText(ais[Ai1].Name + ": R" + strconv.Itoa(intai1Score)).Update()
								ai1Miss = false
							}

						} else {
							intMoveAmount := ChanceCards[randCCard].Value - intai1Pos
							if intMoveAmount < 1 {
								intMoveAmount += 20
							}
							intScoreGain = Move(intai1Pos, intMoveAmount, 2, &pg.Frame, ai1Token)
							intai1Pos = ChanceCards[randCCard].Value 
							if intScoreGain != 0 && intai1Pos != 6 {
								intai1Score += intScoreGain
							}

							randpercentChance := r.Intn(100) + 1
							if ais[Ai1].StatsSection5 >= randpercentChance {
								intai1Score += 100
								txtai1Score.SetText(ais[Ai1].Name + ": R" + strconv.Itoa(intai1Score)).Update()
								ai1Miss = false
							} else {
								ai1Miss = true
							}
						}
					case 6:
						ai1Miss = true
						randpercentChance := r.Intn(100) + 1
						if ais[Ai1].StatsSection5 >= randpercentChance {
							intai1Score += 100
							txtai1Score.SetText(ais[Ai1].Name + ": R" + strconv.Itoa(intai1Score)).Update()
							ai1Miss = false
						}
					case 7, 9, 10:
						randpercentChance := r.Intn(100) + 1
						if ais[Ai1].StatsSection2 >= randpercentChance {
							intai1Score += 100
							txtai1Score.SetText(ais[Ai1].Name + ": R" + strconv.Itoa(intai1Score)).Update()
							ai1Miss = false
						}
					case 11:
						randpercentChance := r.Intn(100) + 1
						if ais[Ai1].StatsSection5 >= randpercentChance {
							intai1Score += 100
							txtai1Score.SetText(ais[Ai1].Name + ": R" + strconv.Itoa(intai1Score)).Update()
							ai1Miss = false
						}
					case 12, 13, 15:
						randpercentChance := r.Intn(100) + 1
						if ais[Ai1].StatsSection3 >= randpercentChance {
							intai1Score += 100
							txtai1Score.SetText(ais[Ai1].Name + ": R" + strconv.Itoa(intai1Score)).Update()
							ai1Miss = false
						}
					case 16:
						randpercentChance := r.Intn(100) + 1
						if ais[Ai1].StatsSection5 >= randpercentChance {
							intai1Score += 100
							txtai1Score.SetText(ais[Ai1].Name + ": R" + strconv.Itoa(intai1Score)).Update()
							ai1Miss = false
						}
					case 17, 19, 20:
						randpercentChance := r.Intn(100) + 1
						if ais[Ai1].StatsSection4 >= randpercentChance {
							intai1Score += 100
							txtai1Score.SetText(ais[Ai1].Name + ": R" + strconv.Itoa(intai1Score)).Update()
							ai1Miss = false
						}
					}

				}
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//AI2
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
				if ai2Miss {
					randpercentChance := r.Intn(100) + 1
					if ais[Ai2].StatsSection5 >= randpercentChance {
						intai2Score += 100
						txtai2Score.SetText(ais[Ai2].Name + ": R" + strconv.Itoa(intai2Score)).Update()
						ai2Miss = false
					}
				} else {
					intRoll := r.Intn(6) + 1
					i := intRoll
					intScoreGain := Move(intai2Pos, i, 3, &pg.Frame, ai2Token)
					intai2Pos += intRoll
					if intScoreGain != 0 {
						intai2Score += intScoreGain
						intai2Pos -= 20
					}
					switch intai2Pos {
					case 1:
						randpercentChance := r.Intn(100) + 1
						if ais[Ai2].StatsSection5 >= randpercentChance {
							intai2Score += 100
							txtai2Score.SetText(ais[Ai2].Name + ": R" + strconv.Itoa(intai2Score)).Update()
							ai2Miss = false
						}
					case 2, 3, 5:
						randpercentChance := r.Intn(100) + 1
						if ais[Ai2].StatsSection1 >= randpercentChance {
							intai2Score += 100
							txtai2Score.SetText(ais[Ai2].Name + ": R" + strconv.Itoa(intai2Score)).Update()
							ai2Miss = false
						}
					case 4, 8, 14, 18:
						if len(ChanceCards) <= 0 {
							log.Fatalf("ChanceCards <= 0")
						}
						randCCard := r.Intn(len(ChanceCards))
						if ChanceCards[randCCard].IsScoreChange {

							intai2Score += ChanceCards[randCCard].Value
							txtai2Score.SetText(ais[Ai2].Name + ": R" + strconv.Itoa(intai2Score)).Update()

						} else if !ChanceCards[randCCard].IsToFieldMove {

							if ChanceCards[randCCard].IsPosChange {
								i = ChanceCards[randCCard].Value
								
								intScoreGain = Move(intai2Pos, i, 3, &pg.Frame, ai2Token)
								intai2Pos += i
								if intScoreGain != 0 {
									intai2Score += intScoreGain
									intai2Pos -= 20
								}
							} else {
								i = ChanceCards[randCCard].Value
								RevMove(intai2Pos, i, 3, &pg.Frame, ai2Token)
								intai2Pos -= i
								if intai2Pos < 1 {
									intai2Pos += 20
								}
							}
							randpercentChance := r.Intn(100) + 1
							if ais[Ai2].StatsSection5 >= randpercentChance {
								intai2Score += 100
								txtai2Score.SetText(ais[Ai2].Name + ": R" + strconv.Itoa(intai2Score)).Update()
								ai2Miss = false
							}
						} else {
							intMoveAmount := ChanceCards[randCCard].Value - intai2Pos
							if intMoveAmount < 1 {
								intMoveAmount += 20
							}
							intScoreGain = Move(intai2Pos, intMoveAmount, 3, &pg.Frame, ai2Token)
							if intScoreGain != 0 && intai2Pos != 6 {
								intai2Score += intScoreGain
							}

							randpercentChance := r.Intn(100) + 1
							if ais[Ai2].StatsSection5 >= randpercentChance {
								intai2Score += 100
								txtai2Score.SetText(ais[Ai2].Name + ": R" + strconv.Itoa(intai2Score)).Update()
								ai2Miss = false
							}
						}
					case 6:
						ai2Miss = true
						randpercentChance := r.Intn(100) + 1
						if ais[Ai2].StatsSection5 >= randpercentChance {
							intai2Score += 100
							txtai2Score.SetText(ais[Ai2].Name + ": R" + strconv.Itoa(intai2Score)).Update()
							ai2Miss = false
						}
					case 7, 9, 10:
						randpercentChance := r.Intn(100) + 1
						if ais[Ai2].StatsSection2 >= randpercentChance {
							intai2Score += 100
							txtai2Score.SetText(ais[Ai2].Name + ": R" + strconv.Itoa(intai2Score)).Update()
							ai2Miss = false
						}
					case 11:
						randpercentChance := r.Intn(100) + 1
						if ais[Ai2].StatsSection5 >= randpercentChance {
							intai2Score += 100
							txtai2Score.SetText(ais[Ai2].Name + ": R" + strconv.Itoa(intai2Score)).Update()
							ai2Miss = false
						}
					case 12, 13, 15:
						randpercentChance := r.Intn(100) + 1
						if ais[Ai2].StatsSection3 >= randpercentChance {
							intai2Score += 100
							txtai2Score.SetText(ais[Ai2].Name + ": R" + strconv.Itoa(intai2Score)).Update()
							ai2Miss = false
						}
					case 16:
						randpercentChance := r.Intn(100) + 1
						if ais[Ai2].StatsSection5 >= randpercentChance {
							intai2Score += 100
							txtai2Score.SetText(ais[Ai2].Name + ": R" + strconv.Itoa(intai2Score)).Update()
							ai2Miss = false
						}
					case 17, 19, 20:
						randpercentChance := r.Intn(100) + 1
						if ais[Ai2].StatsSection4 >= randpercentChance {
							intai2Score += 100
							txtai2Score.SetText(ais[Ai2].Name + ": R" + strconv.Itoa(intai2Score)).Update()
							ai2Miss = false
						}
					}
				}
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//AI3
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
				if ai3Miss {
					randpercentChance := r.Intn(100) + 1
					if ais[Ai3].StatsSection5 >= randpercentChance {
						intai3Score += 100
						txtai3Score.SetText(ais[Ai3].Name + ": R" + strconv.Itoa(intai3Score)).Update()
						ai3Miss = false
					}
				} else {
					intRoll := r.Intn(6) + 1
					i := intRoll
					intScoreGain := Move(intai3Pos, i, 4, &pg.Frame, ai3Token)
					intai3Pos += intRoll
					if intScoreGain != 0 {
						intai3Score += intScoreGain
						intai3Pos -= 20
					}
					switch intai3Pos {
					case 1:
						randpercentChance := r.Intn(100) + 1
						if ais[Ai3].StatsSection5 >= randpercentChance {
							intai3Score += 100
							txtai3Score.SetText(ais[Ai3].Name + ": R" + strconv.Itoa(intai3Score)).Update()
							ai3Miss = false
						}
					case 2, 3, 5:
						randpercentChance := r.Intn(100) + 1
						if ais[Ai3].StatsSection1 >= randpercentChance {
							intai3Score += 100
							txtai3Score.SetText(ais[Ai3].Name + ": R" + strconv.Itoa(intai3Score)).Update()
							ai3Miss = false
						}
					case 4, 8, 14, 18:
						if len(ChanceCards) <= 0 {
							log.Fatalf("ChanceCards <= 0")
						}
						randCCard := r.Intn(len(ChanceCards))
						if ChanceCards[randCCard].IsScoreChange {

							intai3Score += ChanceCards[randCCard].Value
							txtai3Score.SetText(ais[Ai3].Name + ": R" + strconv.Itoa(intai3Score)).Update()

						} else if !ChanceCards[randCCard].IsToFieldMove {

							if ChanceCards[randCCard].IsPosChange {
								i = ChanceCards[randCCard].Value
								
								intScoreGain = Move(intai3Pos, i, 4, &pg.Frame, ai3Token)
								intai3Pos += i
								if intScoreGain != 0 {
									intai3Score += intScoreGain
									intai3Pos -= 20
								}
							} else {
								i = ChanceCards[randCCard].Value
								RevMove(intai3Pos, i, 4, &pg.Frame, ai3Token)
								intai3Pos -= i
								if intai3Pos < 1 {
									intai3Pos += 20
								}
							}
							randpercentChance := r.Intn(100) + 1
							if ais[Ai3].StatsSection5 >= randpercentChance {
								intai3Score += 100
								txtai3Score.SetText(ais[Ai3].Name + ": R" + strconv.Itoa(intai3Score)).Update()
								ai3Miss = false
							}
						} else {
							intMoveAmount := ChanceCards[randCCard].Value - intai3Pos
							if intMoveAmount < 1 {
								intMoveAmount += 20
							}
							intScoreGain = Move(intai3Pos, intMoveAmount, 4, &pg.Frame, ai3Token)
							if intScoreGain != 0 && intai3Pos != 6 {
								intai3Score += intScoreGain
							}

							randpercentChance := r.Intn(100) + 1
							if ais[Ai3].StatsSection5 >= randpercentChance {
								intai3Score += 100
								txtai3Score.SetText(ais[Ai3].Name + ": R" + strconv.Itoa(intai3Score)).Update()
								ai3Miss = false
							}
						}
					case 6:
						ai3Miss = true
						randpercentChance := r.Intn(100) + 1
						if ais[Ai3].StatsSection5 >= randpercentChance {
							intai3Score += 100
							txtai3Score.SetText(ais[Ai3].Name + ": R" + strconv.Itoa(intai3Score)).Update()
							ai3Miss = false
						}
					case 7, 9, 10:
						randpercentChance := r.Intn(100) + 1
						if ais[Ai3].StatsSection2 >= randpercentChance {
							intai3Score += 100
							txtai3Score.SetText(ais[Ai3].Name + ": R" + strconv.Itoa(intai3Score)).Update()
							ai3Miss = false
						}
					case 11:
						randpercentChance := r.Intn(100) + 1
						if ais[Ai3].StatsSection5 >= randpercentChance {
							intai3Score += 100
							txtai3Score.SetText(ais[Ai3].Name + ": R" + strconv.Itoa(intai3Score)).Update()
							ai3Miss = false
						}
					case 12, 13, 15:
						randpercentChance := r.Intn(100) + 1
						if ais[Ai3].StatsSection3 >= randpercentChance {
							intai3Score += 100
							txtai3Score.SetText(ais[Ai3].Name + ": R" + strconv.Itoa(intai3Score)).Update()
							ai3Miss = false
						}
					case 16:
						randpercentChance := r.Intn(100) + 1
						if ais[Ai3].StatsSection5 >= randpercentChance {
							intai3Score += 100
							txtai3Score.SetText(ais[Ai3].Name + ": R" + strconv.Itoa(intai3Score)).Update()
							ai3Miss = false
						}
					case 17, 19, 20:
						randpercentChance := r.Intn(100) + 1
						if ais[Ai3].StatsSection4 >= randpercentChance {
							intai3Score += 100
							txtai3Score.SetText(ais[Ai3].Name + ": R" + strconv.Itoa(intai3Score)).Update()
							ai3Miss = false
						}
					}
				}
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//PLAYER
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////				
				var intRoll int
				if playerMiss {
					intRoll = 0
				} else {
					intRoll = r.Intn(6) + 1
				}
				i := intRoll
				intScoreGain := Move(intplayerPos, i, 1, &pg.Frame, playerToken)
				intplayerPos += intRoll
				if intScoreGain != 0 {
					playerScore += intScoreGain
					intplayerPos -= 20
				}
				switch intplayerPos {
				case 1:
					questionPopup(lQs, &pg.Frame, false, true)
				case 2, 3, 5:
					questionPopup(mmm, &pg.Frame, false, false)
				case 4, 8, 14, 18:
					randCCard := r.Intn(len(ChanceCards))
					Card := ChanceCards[randCCard]
					tacCard := core.NewBody("Chance")
					core.NewText(tacCard).SetText(Card.Message)
					tacCard.AddBottomBar(func(bar *core.Frame) {
						tacCard.AddOK(bar)
						if Card.IsScoreChange {
							if Card.IsPosChange {
								playerScore += Card.Value
							} else {
								playerScore -= Card.Value
							}
						} else {
							if !Card.IsToFieldMove {
								if Card.IsPosChange {
									intScoreGain = Move(intplayerPos, Card.Value, 1, &pg.Frame, playerToken)
									intplayerPos += Card.Value
									if intScoreGain != 0 {
										playerScore += intScoreGain
										intplayerPos -=20
									}
								} else {
									RevMove(intplayerPos, Card.Value, 1, &pg.Frame, playerToken)
									intplayerPos -= Card.Value
									if intplayerPos < 1 {
										intplayerPos += 20
									}
								}
							} else {
								intMoveAmount := Card.Value - intplayerPos
								if intMoveAmount < 1 {
									intMoveAmount += 20
								}
								intScoreGain = Move(intplayerPos, intMoveAmount, 1, &pg.Frame, playerToken)
								intplayerPos = Card.Value
								if intScoreGain != 0 && intplayerPos != 6{
									playerScore += intScoreGain
								}
							}
							switch intplayerPos {
							case 1:
								questionPopup(lQs, &pg.Frame, false, true)
							case 2, 3, 5:
								questionPopup(mmm, &pg.Frame, false, false)
							case 6:
								questionPopup(de, &pg.Frame, true, false)
							case 7, 9, 10:
								questionPopup(ccc, &pg.Frame, false, false)
							case 11:
								questionPopup(co, &pg.Frame, false, false)
							case 12, 13, 15:
								questionPopup(lbl, &pg.Frame, false, false)
							case 16:
								questionPopup(lol, &pg.Frame, false, false)
							case 17, 19, 20:
								questionPopup(ttt, &pg.Frame, false, false)
							}
							txtplayerScore.Update()
						}
					})
					tacCard.RunDialog(&pg.Frame)
				case 6:
					playerMiss = true
					questionPopup(de, &pg.Frame, true, false)
				case 7, 9, 10:
					questionPopup(ccc, &pg.Frame, false, false)
				case 11:
					questionPopup(co, &pg.Frame, false, false)
				case 12, 13, 15:
					questionPopup(lbl, &pg.Frame, false, false)
				case 16:
					questionPopup(lol, &pg.Frame, false, false)
				case 17, 19, 20:
					questionPopup(ttt, &pg.Frame, false, false)
				default:
					core.MessageSnackbar(&pg.Frame, "Error pos out of range: "+strconv.Itoa(intplayerPos))
				}

				if intRoll != 0 {
					diceRoll := core.NewBody("Roll")
					svgDieFace := core.NewSVG(diceRoll)
					switch intRoll {
					case 1:
						err = svgDieFace.OpenFS(embeddedFiles, "Assets/Images/DiceFace1.svg")
					case 2:
						err = svgDieFace.OpenFS(embeddedFiles, "Assets/Images/DiceFace2.svg")
					case 3:
						err = svgDieFace.OpenFS(embeddedFiles, "Assets/Images/DiceFace3.svg")
					case 4:
						err = svgDieFace.OpenFS(embeddedFiles, "Assets/Images/DiceFace4.svg")
					case 5:
						err = svgDieFace.OpenFS(embeddedFiles, "Assets/Images/DiceFace5.svg")
					case 6:
						err = svgDieFace.OpenFS(embeddedFiles, "Assets/Images/DiceFace6.svg")
					}
					if err != nil {
						log.Printf("Error loading file: %s", err)
					}
					diceRoll.AddOKOnly()
					diceRoll.RunDialog(&pg.Frame)
				}

				txtplayerScore.Update()
				Session.Score = int64(playerScore)
			
				//exit logic, session data upload & relevant user data update
				switch GameMode {
				case "Easy":
					if playerScore >= 1500 || intai1Score >= 1500 || intai2Score >= 1500 || intai3Score >= 1500 {
						if playerScore > intai1Score && playerScore > intai2Score && playerScore > intai3Score {
							diaWin := core.NewBody("Game Over")
							core.NewText(diaWin).SetText("You WIN!!")
							diaWin.AddOKOnly()
							diaWin.RunDialog(btnRoll)
						} else {
							if intai1Score > intai2Score && intai1Score > intai3Score {
								diaWin := core.NewBody("Game Over")
								core.NewText(diaWin).SetText(Session.Ai1 + "won")
								diaWin.AddOKOnly()
								diaWin.RunDialog(btnRoll)
							} else if intai2Score > intai3Score {
								diaWin := core.NewBody("Game Over")
								core.NewText(diaWin).SetText(Session.Ai2 + "won")
								diaWin.AddOKOnly()
								diaWin.RunDialog(btnRoll)
							} else {
								diaWin := core.NewBody("Game Over")
								core.NewText(diaWin).SetText(Session.Ai3 + "won")
								diaWin.AddOKOnly()
								diaWin.RunDialog(btnRoll)
							}
						}
						timeNow := time.Now()
						numNanosec := timeNow.Sub(startTime).Nanoseconds()
						CurrentUser.TotalTime += numNanosec
						CurrentUser.TotalScore += Session.Score
						err = MBSEGames.AddSession(ctx, firestoreClient, CurrentUser.DisplayName, Session.Subject, Session.Gamemode, Session.Ai1, Session.Ai2, Session.Ai3, Session.StartTime, timeNow.Format(time.RFC822), Session.QDetails, Session.Score, Session.NumCorrect, Session.NumIncorrect)
						if err != nil {
							log.Fatalf("error adding session: %s", err)
						}
					
						_, nMonth, nDay := timeNow.Date()
						_, lMonth, lDay := CurrentUser.LastPlayed.Date()
						timeBetween := timeNow.Sub(CurrentUser.LastPlayed).Hours()
						bStreakAdd := false
						if nMonth != lMonth && nMonth - lMonth == 1 && nDay == 1 {	
							switch nMonth {
							case 1:
								if lDay == 31 {
									bStreakAdd = true
								}
							case 2:
								if lDay == 28 {
									bStreakAdd = true
								}
							case 3:
								if lDay == 31 {
									bStreakAdd = true
								}
							case 4:
								if lDay == 30 {
									bStreakAdd = true
								}
							case 5:
								if lDay == 31 {
									bStreakAdd = true
								}
							case 6:
								if lDay == 30 {
									bStreakAdd = true
								}
							case 7:
								if lDay == 31 {
									bStreakAdd = true
								}
							case 8:
								if lDay == 31 {
									bStreakAdd = true
								}
							case 9:
								if lDay == 30 {
									bStreakAdd = true
								}
							case 10:
								if lDay == 31 {
									bStreakAdd = true
								}
							case 11:
								if lDay == 30 {
									bStreakAdd = true
								}
							case 12:
								if lDay == 31 {
									bStreakAdd = true
								} 
							}
						} else if nDay - lDay == 1 && nMonth == lMonth {
							bStreakAdd = true
						}
						if bStreakAdd {
							CurrentUser.Streak++
							err, _ = MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.DocRef, "Streak", "", CurrentUser.Streak, 0)
							if err != nil {
								log.Fatalf("error updating streak: %s", err)
							}
							if CurrentUser.Streak > CurrentUser.RecordStreak {
								err, _ = MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.DocRef, "RecordStreak", "", CurrentUser.Streak, 0)
								if err != nil {
									log.Fatalf("error updating record streak: %s", err)
								}
							}
						} else if timeBetween > 48 {
							CurrentUser.Streak = 1
							err, _ = MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.DocRef, "Streak", "", CurrentUser.Streak, 0)
							if err != nil {
								log.Fatalf("error updating streak: %s", err)
							}
						}

						err, _ = MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.DocRef, "LastDate", timeNow.Format(time.RFC3339), 0, 0)
						if err != nil {
							log.Fatalf("error updating last date: %s", err)
						}

						err, _ = MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.DocRef, "TotalTime", "", 0, CurrentUser.TotalTime)
						if err != nil {
							log.Fatalf("error updating total time: %s", err)
						}

						err, _ = MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.DocRef, "TotalScore", "", 0, CurrentUser.TotalScore)
						if err != nil {
							log.Fatalf("error updating total score: %s", err)
						}

						pagesWidget.Open("Player")
					}

				case "Medium":
					if playerScore >= 3000 || intai1Score >= 3000 || intai2Score >= 3000 || intai3Score >= 3000 {
						if playerScore > intai1Score && playerScore > intai2Score && playerScore > intai3Score {
							diaWin := core.NewBody("Game Over")
							core.NewText(diaWin).SetText("You WIN!!")
							diaWin.AddOKOnly()
							diaWin.RunDialog(btnRoll)
						} else {
							if intai1Score > intai2Score && intai1Score > intai3Score {
								diaWin := core.NewBody("Game Over")
								core.NewText(diaWin).SetText(Session.Ai1 + "won")
								diaWin.AddOKOnly()
								diaWin.RunDialog(btnRoll)
							} else if intai2Score > intai3Score {
								diaWin := core.NewBody("Game Over")
								core.NewText(diaWin).SetText(Session.Ai2 + "won")
								diaWin.AddOKOnly()
								diaWin.RunDialog(btnRoll)
							} else {
								diaWin := core.NewBody("Game Over")
								core.NewText(diaWin).SetText(Session.Ai3 + "won")
								diaWin.AddOKOnly()
								diaWin.RunDialog(btnRoll)
							}
						}
						timeNow := time.Now()
						numNanosec := timeNow.Sub(startTime).Nanoseconds()
						CurrentUser.TotalTime += numNanosec
						CurrentUser.TotalScore += Session.Score
						err = MBSEGames.AddSession(ctx, firestoreClient, CurrentUser.DisplayName, Session.Subject, Session.Gamemode, Session.Ai1, Session.Ai2, Session.Ai3, Session.StartTime, timeNow.Format(time.RFC822), Session.QDetails, Session.Score, Session.NumCorrect, Session.NumIncorrect)
						if err != nil {
							log.Fatalf("error adding session: %s", err)
						}
					
						_, nMonth, nDay := timeNow.Date()
						_, lMonth, lDay := CurrentUser.LastPlayed.Date()
						timeBetween := timeNow.Sub(CurrentUser.LastPlayed).Hours()
						bStreakAdd := false
						if nMonth != lMonth && nMonth - lMonth == 1 && nDay == 1 {	
							switch nMonth {
							case 1:
								if lDay == 31 {
									bStreakAdd = true
								}
							case 2:
								if lDay == 28 {
									bStreakAdd = true
								}
							case 3:
								if lDay == 31 {
									bStreakAdd = true
								}
							case 4:
								if lDay == 30 {
									bStreakAdd = true
								}
							case 5:
								if lDay == 31 {
									bStreakAdd = true
								}
							case 6:
								if lDay == 30 {
									bStreakAdd = true
								}
							case 7:
								if lDay == 31 {
									bStreakAdd = true
								}
							case 8:
								if lDay == 31 {
									bStreakAdd = true
								}
							case 9:
								if lDay == 30 {
									bStreakAdd = true
								}
							case 10:
								if lDay == 31 {
									bStreakAdd = true
								}
							case 11:
								if lDay == 30 {
									bStreakAdd = true
								}
							case 12:
								if lDay == 31 {
									bStreakAdd = true
								} 
							}
						} else if nDay - lDay == 1 && nMonth == lMonth {
							bStreakAdd = true
						}
						if bStreakAdd {
							CurrentUser.Streak++
							err, _ = MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.DocRef, "Streak", "", CurrentUser.Streak, 0)
							if err != nil {
								log.Fatalf("error updating streak: %s", err)
							}
							if CurrentUser.Streak > CurrentUser.RecordStreak {
								err, _ = MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.DocRef, "RecordStreak", "", CurrentUser.Streak, 0)
								if err != nil {
									log.Fatalf("error updating record streak: %s", err)
								}
							}
						} else if timeBetween > 48 {
							CurrentUser.Streak = 1
							err, _ = MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.DocRef, "Streak", "", CurrentUser.Streak, 0)
							if err != nil {
								log.Fatalf("error updating streak: %s", err)
							}
						}

						err, _ = MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.DocRef, "LastDate", timeNow.Format(time.RFC3339), 0, 0)
						if err != nil {
							log.Fatalf("error updating last date: %s", err)
						}

						err, _ = MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.DocRef, "TotalTime", "", 0, CurrentUser.TotalTime)
						if err != nil {
							log.Fatalf("error updating total time: %s", err)
						}

						err, _ = MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.DocRef, "TotalScore", "", 0, CurrentUser.TotalScore)
						if err != nil {
							log.Fatalf("error updating total score: %s", err)
						}

						pagesWidget.Open("Player")
					}
				case "Hard":
					if playerScore >= 5000 || intai1Score >= 5000 || intai2Score >= 5000 || intai3Score >= 5000 {
						if playerScore > intai1Score && playerScore > intai2Score && playerScore > intai3Score {
							diaWin := core.NewBody("Game Over")
							core.NewText(diaWin).SetText("You WIN!!")
							diaWin.AddOKOnly()
							diaWin.RunDialog(btnRoll)
						} else {
							if intai1Score > intai2Score && intai1Score > intai3Score {
								diaWin := core.NewBody("Game Over")
								core.NewText(diaWin).SetText(Session.Ai1 + "won")
								diaWin.AddOKOnly()
								diaWin.RunDialog(btnRoll)
							} else if intai2Score > intai3Score {
								diaWin := core.NewBody("Game Over")
								core.NewText(diaWin).SetText(Session.Ai2 + "won")
								diaWin.AddOKOnly()
								diaWin.RunDialog(btnRoll)
							} else {
								diaWin := core.NewBody("Game Over")
								core.NewText(diaWin).SetText(Session.Ai3 + "won")
								diaWin.AddOKOnly()
								diaWin.RunDialog(btnRoll)
							}
						}
						timeNow := time.Now()
						numNanosec := timeNow.Sub(startTime).Nanoseconds()
						CurrentUser.TotalTime += numNanosec
						CurrentUser.TotalScore += Session.Score
						err = MBSEGames.AddSession(ctx, firestoreClient, CurrentUser.DisplayName, Session.Subject, Session.Gamemode, Session.Ai1, Session.Ai2, Session.Ai3, Session.StartTime, timeNow.Format(time.RFC822), Session.QDetails, Session.Score, Session.NumCorrect, Session.NumIncorrect)
						if err != nil {
							log.Fatalf("error adding session: %s", err)
						}
					
						_, nMonth, nDay := timeNow.Date()
						_, lMonth, lDay := CurrentUser.LastPlayed.Date()
						timeBetween := timeNow.Sub(CurrentUser.LastPlayed).Hours()
						bStreakAdd := false
						if nMonth != lMonth && nMonth - lMonth == 1 && nDay == 1 {	
							switch nMonth {
							case 1:
								if lDay == 31 {
									bStreakAdd = true
								}
							case 2:
								if lDay == 28 {
									bStreakAdd = true
								}
							case 3:
								if lDay == 31 {
									bStreakAdd = true
								}
							case 4:
								if lDay == 30 {
									bStreakAdd = true
								}
							case 5:
								if lDay == 31 {
									bStreakAdd = true
								}
							case 6:
								if lDay == 30 {
									bStreakAdd = true
								}
							case 7:
								if lDay == 31 {
									bStreakAdd = true
								}
							case 8:
								if lDay == 31 {
									bStreakAdd = true
								}
							case 9:
								if lDay == 30 {
									bStreakAdd = true
								}
							case 10:
								if lDay == 31 {
									bStreakAdd = true
								}
							case 11:
								if lDay == 30 {
									bStreakAdd = true
								}
							case 12:
								if lDay == 31 {
									bStreakAdd = true
								} 
							}
						} else if nDay - lDay == 1 && nMonth == lMonth {
							bStreakAdd = true
						}
						if bStreakAdd {
							CurrentUser.Streak++
							err, _ = MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.DocRef, "Streak", "", CurrentUser.Streak, 0)
							if err != nil {
								log.Fatalf("error updating streak: %s", err)
							}
							if CurrentUser.Streak > CurrentUser.RecordStreak {
								err, _ = MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.DocRef, "RecordStreak", "", CurrentUser.Streak, 0)
								if err != nil {
									log.Fatalf("error updating record streak: %s", err)
								}
							}
						} else if timeBetween > 48 {
							CurrentUser.Streak = 1
							err, _ = MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.DocRef, "Streak", "", CurrentUser.Streak, 0)
							if err != nil {
								log.Fatalf("error updating streak: %s", err)
							}
						}

						err, _ = MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.DocRef, "LastDate", timeNow.Format(time.RFC3339), 0, 0)
						if err != nil {
							log.Fatalf("error updating last date: %s", err)
						}

						err, _ = MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.DocRef, "TotalTime", "", 0, CurrentUser.TotalTime)
						if err != nil {
							log.Fatalf("error updating total time: %s", err)
						}

						err, _ = MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.DocRef, "TotalScore", "", 0, CurrentUser.TotalScore)
						if err != nil {
							log.Fatalf("error updating total score: %s", err)
						}

						pagesWidget.Open("Player")
					}
				}
			}()
		})

	})

	b.RunMainWindow()	
	
	// Shutdown the local HTTP server gracefully
	log.Println("Shutting down local server...")
	if err := listener.Close(); err != nil {
		log.Printf("Error closing listener: %v", err)
	} else {
		log.Println("Local server listener closed.")
	}
	time.Sleep(100 * time.Millisecond)
}

func Move(intCurrentPos, intRoll, tokenNum int, bd *core.Frame, token *core.Icon) (intScoreGained int) {
	intScoreGained = 0
	var XVal, YVal units.Value
	XVal.Set(0, units.UnitVh)
	YVal.Set(0, units.UnitVh)
	
	for intRoll > 0 {
		bd.AsyncLock()
		intCurrentPos++
		if intCurrentPos == 21 {
			intCurrentPos = 1
			intScoreGained += 200
		}

		XVal.Value, YVal.Value = MovementLoad(intCurrentPos, tokenNum)
		token.Styler(func(s *styles.Style) {
			s.Pos.Set(XVal, YVal)
		})
		token.Update()
		bd.NeedsLayout()
		bd.AsyncUnlock()

		time.Sleep(200 * time.Millisecond)

		intRoll--
	}

	return intScoreGained
}

func RevMove(intCurrentPos, intRoll, tokenNum int, bd *core.Frame, token *core.Icon) {
	var XVal, YVal units.Value
	XVal.Set(0, units.UnitVh)
	YVal.Set(0, units.UnitVh)
	
	for intRoll > 0 {
		bd.AsyncLock()
		intCurrentPos--
		if intCurrentPos == 0 {
			intCurrentPos = 20
		}

		XVal.Value, YVal.Value = MovementLoad(intCurrentPos, tokenNum)
		token.Styler(func(s *styles.Style) {
			s.Pos.Set(XVal, YVal)
		})
		token.Update()
		bd.NeedsLayout()
		bd.AsyncUnlock()

		time.Sleep(200 * time.Millisecond)

		intRoll--
	}
}

func MovementLoad(intPos, TokenNum int) (float32, float32) {
	var XFloat, YFloat float32
	switch intPos {
	case 1:
		switch TokenNum {
		case 1:
			XFloat = 3.3333333333
			YFloat = 86.6666666667
		case 2:
			XFloat = 10
			YFloat = 86.6666666667
		case 3:
			XFloat = 3.3333333333
			YFloat = 93.3333333333
		case 4:
			XFloat = 10
			YFloat = 93.3333333333
		}

	case 2:
		switch TokenNum {
		case 1:
			XFloat = 3.3333333333
			YFloat = 70
		case 2:
			XFloat = 10
			YFloat = 70
		case 3:
			XFloat = 3.3333333333
			YFloat = 76.6666666667
		case 4:
			XFloat = 10
			YFloat = 76.6666666667
		}

	case 3:
		switch TokenNum {
		case 1:
			XFloat = 3.3333333333
			YFloat = 53.3333333333
		case 2:
			XFloat = 10
			YFloat = 53.3333333333
		case 3:
			XFloat = 3.3333333333
			YFloat = 60
		case 4:
			XFloat = 10
			YFloat = 60
		}

	case 4:
		switch TokenNum {
		case 1:
			XFloat = 3.3333333333
			YFloat = 36.6666666667
		case 2:
			XFloat = 10
			YFloat = 36.6666666667
		case 3:
			XFloat = 3.3333333333
			YFloat = 43.3333333333
		case 4:
			XFloat = 10
			YFloat = 43.3333333333
		}

	case 5:
		switch TokenNum {
		case 1:
			XFloat = 3.3333333333
			YFloat = 20
		case 2:
			XFloat = 10
			YFloat = 20
		case 3:
			XFloat = 3.3333333333
			YFloat = 26.6666666667
		case 4:
			XFloat = 10
			YFloat = 26.6666666667
		}

	case 6:
		switch TokenNum {
		case 1:
			XFloat = 3.3333333333
			YFloat = 3.3333333333
		case 2:
			XFloat = 10
			YFloat = 3.3333333333
		case 3:
			XFloat = 3.3333333333
			YFloat = 10
		case 4:
			XFloat = 10
			YFloat = 10
		}

	case 7:
		switch TokenNum {
		case 1:
			XFloat = 20
			YFloat = 3.3333333333
		case 2:
			XFloat = 26.6666666667
			YFloat = 3.3333333333
		case 3:
			XFloat = 20
			YFloat = 10
		case 4:
			XFloat = 26.6666666667
			YFloat = 10
		}

	case 8:
		switch TokenNum {
		case 1:
			XFloat = 36.6666666667
			YFloat = 3.3333333333
		case 2:
			XFloat = 43.3333333333
			YFloat = 3.3333333333
		case 3:
			XFloat = 36.6666666667
			YFloat = 10
		case 4:
			XFloat = 43.3333333333
			YFloat = 10
		}

	case 9:
		switch TokenNum {
		case 1:
			XFloat = 53.3333333333
			YFloat = 3.3333333333
		case 2:
			XFloat = 60
			YFloat = 3.3333333333
		case 3:
			XFloat = 53.3333333333
			YFloat = 10
		case 4:
			XFloat = 60
			YFloat = 10
		}

	case 10:
		switch TokenNum {
		case 1:
			XFloat = 70
			YFloat = 3.3333333333
		case 2:
			XFloat = 76.6666666667
			YFloat = 3.3333333333
		case 3:
			XFloat = 70
			YFloat = 10
		case 4:
			XFloat = 76.6666666667
			YFloat = 10
		}

	case 11:
		switch TokenNum {
		case 1:
			XFloat = 86.6666666667
			YFloat = 3.3333333333
		case 2:
			XFloat = 93.3333333333
			YFloat = 3.3333333333
		case 3:
			XFloat = 86.6666666667
			YFloat = 10
		case 4:
			XFloat = 93.3333333333
			YFloat = 10
		}

	case 12:
		switch TokenNum {
		case 1:
			XFloat = 86.6666666667
			YFloat = 20
		case 2:
			XFloat = 93.3333333333
			YFloat = 20
		case 3:
			XFloat = 86.6666666667
			YFloat = 26.6666666667
		case 4:
			XFloat = 93.3333333333
			YFloat = 26.6666666667
		}

	case 13:
		switch TokenNum {
		case 1:
			XFloat = 86.6666666667
			YFloat = 36.6666666667
		case 2:
			XFloat = 93.3333333333
			YFloat = 36.6666666667
		case 3:
			XFloat = 86.6666666667
			YFloat = 43.3333333333
		case 4:
			XFloat = 93.3333333333
			YFloat = 43.3333333333
		}

	case 14:
		switch TokenNum {
		case 1:
			XFloat = 86.6666666667
			YFloat = 53.3333333333
		case 2:
			XFloat = 93.3333333333
			YFloat = 53.3333333333
		case 3:
			XFloat = 86.6666666667
			YFloat = 60
		case 4:
			XFloat = 93.3333333333
			YFloat = 60
		}

	case 15:
		switch TokenNum {
		case 1:
			XFloat = 86.6666666667
			YFloat = 70
		case 2:
			XFloat = 93.3333333333
			YFloat = 70
		case 3:
			XFloat = 86.6666666667
			YFloat = 76.6666666667
		case 4:
			XFloat = 93.3333333333
			YFloat = 76.6666666667
		}

	case 16:
		switch TokenNum {
		case 1:
			XFloat = 86.6666666667
			YFloat = 86.6666666667
		case 2:
			XFloat = 93.3333333333
			YFloat = 86.6666666667
		case 3:
			XFloat = 86.6666666667
			YFloat = 93.3333333333
		case 4:
			XFloat = 93.3333333333
			YFloat = 93.3333333333
		}

	case 17:
		switch TokenNum {
		case 1:
			XFloat = 70
			YFloat = 86.6666666667
		case 2:
			XFloat = 76.6666666667
			YFloat = 86.6666666667
		case 3:
			XFloat = 70
			YFloat = 93.3333333333
		case 4:
			XFloat = 76.6666666667
			YFloat = 93.3333333333
		}

	case 18:
		switch TokenNum {
		case 1:
			XFloat = 53.3333333333
			YFloat = 86.6666666667
		case 2:
			XFloat = 60
			YFloat = 86.6666666667
		case 3:
			XFloat = 53.3333333333
			YFloat = 93.3333333333
		case 4:
			XFloat = 60
			YFloat = 93.3333333333
		}

	case 19:
		switch TokenNum {
		case 1:
			XFloat = 36.6666666667
			YFloat = 86.6666666667
		case 2:
			XFloat = 43.3333333333
			YFloat = 86.6666666667
		case 3:
			XFloat = 36.6666666667
			YFloat = 93.3333333333
		case 4:
			XFloat = 43.3333333333
			YFloat = 93.3333333333
		}

	case 20:
		switch TokenNum {
		case 1:
			XFloat = 20
			YFloat = 86.6666666667
		case 2:
			XFloat = 26.6666666667
			YFloat = 86.6666666667
		case 3:
			XFloat = 20
			YFloat = 93.3333333333
		case 4:
			XFloat = 26.6666666667
			YFloat = 93.3333333333
		}
	}

	return XFloat, YFloat
}

func questionPopup(questionSection []MBSEGames.Question, bd *core.Frame, canMiss, isPos1 bool) {
	r := *rand.New(rand.NewSource(time.Now().Unix()))
	ri := r.Intn(len(questionSection))
	strQRef := questionSection[ri].DocRef
	intOutcome := questionSection[ri].Outcome
	strQuestion := questionSection[ri].Question
	strOption1 := questionSection[ri].Option1
	strOption2 := questionSection[ri].Option2
	strOption3 := questionSection[ri].Option3
	strOption4 := questionSection[ri].Option4
	strExp := questionSection[ri].Explanation
	intAns := questionSection[ri].Answer

	var strTitle string
	switch questionSection[ri].Category {
	case "MMM":
		strTitle = "Money Mastery Meander"
	case "CCC":
		strTitle = "Client Connection Corner"
	case "LBL":
		strTitle = "Legal Beagle Lane"
	case "TTT":
		strTitle = "Tax Tactics Terrace"
	case "LOL":
		strTitle = "Have a Laugh"
	default:
		strTitle = questionSection[ri].Category
	}

	diaQuestion := core.NewBody(strTitle)
	frQuestion := core.NewFrame(diaQuestion)
	frQuestion.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.CenterAll()
	})
	core.NewText(frQuestion).SetText(strQuestion)
	var options []core.SwitchItem
	var option core.SwitchItem
	option.Value = 1
	option.Text = strOption1
	option.Tooltip = ""
	options = append(options, option)

	option.Value = 2
	option.Text = strOption2
	option.Tooltip = ""
	options = append(options, option)

	option.Value = 3
	option.Text = strOption3
	option.Tooltip = ""
	options = append(options, option)

	option.Value = 4
	option.Text = strOption4
	option.Tooltip = ""
	options = append(options, option)

	rbgOptions := core.NewSwitches(frQuestion).SetMutex(true).SetType(core.SwitchRadioButton)
	rbgOptions.SetItems(options[0], options[1], options[2], options[3])

	diaQuestion.AddBottomBar(func(bar *core.Frame) {
		diaQuestion.AddOKOnly().AddOK(bar).OnClick(func(e events.Event) {
			selVal := rbgOptions.SelectedItem().Value
			var strAnsGive string
			switch selVal {
			case 1: strAnsGive = "1"
			case 2: strAnsGive = "2"
			case 3: strAnsGive = "3"
			case 4: strAnsGive = "4"
			default: strAnsGive = "0"
			}
			if intAns == selVal {
				Session.NumCorrect++
				if !isPos1 {
					playerScore += intOutcome
				} else {
					playerScore += 400
				}
				strPlayerScoreLbl = MBSEGames.CurrentUser.DisplayName + ": R" + strconv.Itoa(playerScore)
				playerMiss = false
				core.MessageDialog(bar, strExp, "Correct!")
				Session.QDetails += strQRef + "," + strAnsGive + ",1;"
			} else {
				Session.NumIncorrect++
				if canMiss {
					playerMiss = true
				}
				core.MessageDialog(bar, strExp, "Incorrect...")
				Session.QDetails += strQRef + "," + strAnsGive + ",0;"
			}
		})
	})
	diaQuestion.RunDialog(bd)
}

// --- Lorca Login Window ---

func showLoginWindow(serverAddr string) {
	args := []string{}
	// Add custom arguments to Chrome/Edge if needed
	args = append(args, "--remote-allow-origins=*") // May be required sometimes
	ui, err := lorca.New(serverAddr+"/Code/FirebaseAuth/login.html", "", 500, 650, args...)
	if err != nil {
		log.Printf("Failed to create Lorca UI (Is Chrome/Edge installed?): %v", err)
		log.Println(">>> Go: Signaling login completion (WaitGroup Done) due to error.")
		loginWaitGroup.Done() // Signal completion even on error so main doesn't block forever
		return
	}
	// Close UI automatically when function exits
	// Using defer AFTER checking for err ensures we don't try to close a nil ui
	defer ui.Close()

	// --- Bind Go function to be callable from JavaScript ---
	// JS will call: window.goSendToken("the-id-token-string")
	err = ui.Bind("goSendToken", func(token string) {
		log.Println(">>> Go: Received ID Token from JavaScript!")
		if len(token) > 15 {
			log.Printf(">>> Go: Token starts with: %s...\n", token[:15])
		}

		// Store the token
		receivedIDToken = token

		// Signal that the login process is complete
		log.Println(">>> Go: Signaling login completion (WaitGroup Done).")
		loginWaitGroup.Done()

		// Close the login window automatically after getting token
		// No need for Dispatch, Lorca handles thread safety for Close
		log.Println(">>> Go: Closing Lorca window.")
		if closeErr := ui.Close(); closeErr != nil {
			log.Printf("Error closing lorca ui: %v", closeErr)
		}
	})
	if err != nil {
		log.Printf("Failed to bind function: %v", err)
		// Don't signal Done here, let <-ui.Done() handle the window closing
		// loginWaitGroup.Done() // Removed this line from error case
		return // Exit if binding fails
	}

	log.Println("Lorca UI created. Waiting for window to close or token...")

	// --- Wait for the Lorca window to be closed ---
	// This blocks until ui.Close() is called or the user closes the window manually.
	<-ui.Done()

	log.Println("Lorca window closed.")

	// If the window was closed *manually* before login, Done() will be called,
	// but the WaitGroup might not have been signaled yet.
	// To prevent main from waiting forever if the user closes the window early,
	// we attempt to signal Done here as well. This is safe because wg.Done()
	// panics if called more times than wg.Add(). We only Added 1.
	// A more robust solution might use channels or check receivedIDToken's status.
	// For simplicity, we risk a potential panic if ui.Close() in the binding *and*
	// manual close happen nearly simultaneously, OR we accept main might wait
	// if closed manually. Let's try a non-panicking approach:
	// Check if token is still empty; if so, the window was likely closed manually before success.
	if receivedIDToken == "" {
		log.Println("Lorca window closed without receiving token (likely manual close). Signaling Done.")
		// Only signal Done if the binding callback hasn't already done so.
		// A sync.Once could also manage this, but WaitGroup is already here.
		// This simple check isn't perfectly race-proof but covers the main case.
		// A better way: use a dedicated channel for signaling completion.
		loginWaitGroup.Done() // Signal completion anyway
	}
}

// --- Post-Login Handler ---

func handleLoginSuccess(authClient *auth.Client) {
	log.Println("-----------------------------------------")
	log.Println("Login Successful!")
	log.Printf("ID Token stored (starts with: %s...)\n", receivedIDToken[:15])
	log.Println("-----------------------------------------")

	// --- A. Verify the token using Admin SDK ---
	ctx := context.Background()
	verifiedToken, err := authClient.VerifyIDToken(ctx, receivedIDToken)
	if err != nil {
		log.Printf("ERROR verifying ID token: %v\n", err)
		log.Println("Cannot proceed with verified user data.")
		log.Println("-----------------------------------------")
		// Handle verification failure appropriately (e.g., show error in UI, logout)
		return
	}

	// --- B. Get User Info ---
	log.Printf("Token Verified Successfully!")
	log.Printf("User UID: %s\n", verifiedToken.UID)
	if email, ok := verifiedToken.Claims["email"].(string); ok {
		log.Printf("User Email: %s\n", email)
	}
	// Access other standard or custom claims from verifiedToken.Claims map

	log.Println("-----------------------------------------")

	// --- C/D/E: Interact with Firebase Services ---
	currentUserID = verifiedToken.UID
}

// --- Local HTTP Server for Embedded Files (Identical to previous example) ---

func startLocalServer(ctx context.Context, content embed.FS) (net.Listener, string, error) {
	fsys, err := fs.Sub(content, ".")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create sub filesystem: %w", err)
	}
	http.Handle("/", http.FileServer(http.FS(fsys)))

	listener, err := net.Listen("tcp", "localhost:0") // Random port
	if err != nil {
		return nil, "", fmt.Errorf("failed to listen on random port: %w", err)
	}

	serverAddr := fmt.Sprintf("http://%s", listener.Addr().String())

	go func() {
		log.Printf("HTTP server starting on %s", serverAddr)
		server := &http.Server{}
		err := server.Serve(listener)
		if err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server error: %v", err)
		} else {
			log.Println("HTTP server stopped.")
		}
	}()

	return listener, serverAddr, nil
}
