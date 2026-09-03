package main
//.Close()
import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"time"

	MBSEGames "MBSEGames/Code/Database"

	"cloud.google.com/go/firestore"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/system"
	firebase "firebase.google.com/go/v4" // Added Firebase Admin SDK import
)

//go:embed Assets/Images/Moonopoly.png Assets/Images/DiceFace1.svg Assets/Images/DiceFace2.svg Assets/Images/DiceFace3.svg Assets/Images/DiceFace4.svg Assets/Images/DiceFace5.svg Assets/Images/DiceFace6.svg Assets/Images/BackgroundV3.png
var embeddedFiles embed.FS

// Global variable to store the received token
var currentUseremail, glbltoken string
var firestoreClient *firestore.Client
var app *firebase.App
var playerScore int
var strPlayerScoreLbl string
var playerMiss bool
var ctx context.Context
var Ai1, Ai2, Ai3 int
var GameMode string
var startTime time.Time
var leadScores []scoreLead
var leadStreaks []streakLead
var leadTimes []timeLead
var Users []MBSEGames.User
var Questions []MBSEGames.Question
var lQs, ccc, mmm, lbl, lol, ttt, de, co []MBSEGames.Question
var ChanceCards []MBSEGames.ChanceCard
var ai AI
var CurrentUser MBSEGames.User
var CUserName string
var sec, min, hour, days int
var disTL, disScL, disStL []disLead
var intplayerPos, intai1Pos, intai2Pos, intai3Pos int

type AI struct {
	Name          string
	StatsSection1 int
	StatsSection2 int
	StatsSection3 int
	StatsSection4 int
	StatsSection5 int
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

type LoginPayload struct {
	Email             string `json:"email"`
	Password          string `json:"password"`
	ReturnSecureToken bool   `json:"returnSecureToken"`
}

type authResponse struct {
	IdToken string `json:"idToken"`
	Email   string `json:"email"`
}

type FirestoreDoc struct {
	Name   string                 `json:"name"`
	Fields map[string]interface{} `json:"fields"`
}

var ais []AI
var Session MBSEGames.Session

// --- Main Application Logic ---

func main() {
	intplayerPos = 1
	intai1Pos = 1
	intai2Pos = 1
	intai3Pos = 1
	b := core.NewBody()
	b.Styler(func(s *styles.Style) {
		s.Background = colors.Uniform(colors.FromRGB(18, 19, 22))
	})

	pagesWidget := core.NewPages(b).SetPage("Login")
	pagesWidget.AddPage("Login", func(pg *core.Pages) {
		lblHead := core.NewText(pg).SetType(core.TextHeadlineMedium).SetText("Authentication")
		lblSubHead := core.NewText(pg).SetText("You must log in to play.")
		lblHead.Styler(func(s *styles.Style) {
			s.Color = colors.Uniform(colors.FromRGB(225, 227, 230))
		})
		lblSubHead.Styler(func(s *styles.Style) {
			s.Color = colors.Uniform(colors.FromRGB(225, 227, 230))
		})
		emailIn := core.NewTextField(pg).SetPlaceholder("Email")
		passIn := core.NewTextField(pg).SetPlaceholder("Password").SetTypePassword()
		statusLbl := core.NewText(pg).SetText("")
		statusLbl.Styler(func(s *styles.Style) {
			s.Color = colors.Scheme.Error.Base
		})

		// 4. Login Button Logic
		btn := core.NewButton(pg).SetText("Log In")
		btn.OnClick(func(e events.Event) {
			email := emailIn.Text()
			pass := passIn.Text()
			statusLbl.SetText("Verifying...")
			pg.Update()

			go func() {
				// Perform your REST Login (using the code we wrote before)
				token, cuemail, err := restLogin(email, pass)
				glbltoken = token
				currentUseremail = cuemail
				if err != nil {
					statusLbl.SetText("Error: " + err.Error())
					pg.Update()
				} else {
					// SUCCESS!

					ctx = context.Background()
					// We use the new adapter function here
					locapp, client, err := MBSEGames.InitializeFirebaseWithUser(ctx, glbltoken, MBSEGames.ProjectID)
					if err != nil {
						log.Fatalf("SDK Error: %s", err)
						return
					}

					// Assign to your globals
					firestoreClient = client
					app = locapp
					handleLoginSuccess(glbltoken, currentUseremail)

					err = MBSEGames.Login(ctx, app, firestoreClient, currentUseremail)
					if err != nil {
						log.Printf("error finding user '%s', error message: %v", currentUseremail, err)
					}

					CurrentUser = MBSEGames.CurrentUser
					CUserName = CurrentUser.DisplayName

					err = MBSEGames.GetUsers(ctx, firestoreClient)
					if err != nil {
						log.Printf("error getting Clients: %s", err)
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
					pagesWidget.Update()
					pagesWidget.Open("Player")
				}
			}()
		})
	})
	
	/////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	//////Player Home Page
	/////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	pagesWidget.AddPage("Player", func(pg *core.Pages) {
		img, _, err := imagex.OpenFS(embeddedFiles, "Assets/Images/BackgroundV3.png")
		imgBackground := imagex.Resize(img, image.Point{2400, 2400})
		if err != nil {
			log.Fatalf("Error loading image: %s", err)
		}

		pg.Frame.Styler(func(s *styles.Style) {
			s.Background = imgBackground
			s.ObjectFit = styles.FitCover
		})
		playerTabs := core.NewTabs(pg)
		frpHome, btnpHome := playerTabs.NewTab("Home")
		btnpHome.SetIcon(icons.Home)
		frpHome.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.CenterAll()
		})
		lblWel := core.NewText(frpHome).SetText("Welcome " + CUserName + "!")
		lblWel.Styler(func(s *styles.Style) {
			s.Color = colors.Uniform(colors.FromRGB(225, 227, 230))
		})

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
		lblStreak := core.NewText(frpHome).SetText("Current Streak " + strconv.Itoa(CurrentUser.Streak) + "!")
		lblStreak.Styler(func(s *styles.Style) {
			s.Color = colors.Uniform(colors.FromRGB(225, 227, 230))
		})
		lblScore := core.NewText(frpHome).SetText("Total Score " + strconv.Itoa(int(CurrentUser.TotalScore)) + "!")
		lblScore.Styler(func(s *styles.Style) {
			s.Color = colors.Uniform(colors.FromRGB(225, 227, 230))
		})
		nanSec := CurrentUser.TotalTime
		sec := nanSec / 1000000000
		min := sec / 60
		hour := min / 60
		days := hour / 24
		strTime := "You've played for a total of " + strconv.FormatInt(days, 10) + " day/s " + strconv.FormatInt(hour, 10) + " hour/s " + strconv.FormatInt(min, 10) + " minute/s and " + strconv.FormatInt(sec, 10) + " second/s!"
		lblTime := core.NewText(frpHome).SetText(strTime)
		lblTime.Styler(func(s *styles.Style) {
			s.Color = colors.Uniform(colors.FromRGB(225, 227, 230))
		})
		//Leaderboards
		frpLead, btnpLead := playerTabs.NewTab("Leaderboards")
		btnpLead.SetIcon(icons.SocialLeaderboard)
		frpLead.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Gap.Y.Set(5, units.UnitVh)
			s.Grow.Set(1, 1)
			s.Background = colors.Uniform(color.Transparent)
			s.CenterAll()
		})

		frScore := core.NewFrame(frpLead)
		frScore.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Min.Set(units.Vw(100), units.Vh(45))
		})
		core.NewText(frScore).SetText("Score Leaders")
		core.NewTable(frScore).SetSlice(&disScL).SetReadOnly(true)
		
		frStreak := core.NewFrame(frpLead)
		frStreak.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Min.Set(units.Vw(100), units.Vh(45))
		})
		core.NewText(frStreak).SetText("Streak Leaders")
		core.NewTable(frStreak).SetSlice(&disStL).SetReadOnly(true)
		
		frTime := core.NewFrame(frpLead)
		frTime.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Min.Set(units.Vw(100), units.Vh(45))
		})
		core.NewText(frTime).SetText("Time Leaders")
		core.NewTable(frTime).SetSlice(&disTL).SetReadOnly(true)

		//Setup Page
		frPlay, btnPlay := playerTabs.NewTab("Play")
		btnPlay.SetIcon(icons.PlayCircle)
		frPlay.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.CenterAll()
		})
		lblPlay := core.NewText(frPlay).SetText("Press the button below to get started!")
		lblPlay.Styler(func(s *styles.Style) {
			s.Color = colors.Uniform(colors.FromRGB(225, 227, 230))
		})
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
		boardSize.X = winSize.X
		boardSize.Y = winSize.X
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
			s.Min.Set(units.Pw(35), units.Ph(10))
			s.Max.Set(units.Pw(35), units.Ph(10))
			s.Pos.Set(units.Pw(10), units.Ph(50))
			s.RenderBox = false
			s.Background = colors.Uniform(colors.FromRGB(18, 19, 22))
		})
		playerTokenID := core.NewIcon(frplayerScore)
		playerTokenID.Styler(func(s *styles.Style) {
			s.IconSize.Set(units.Vw(5))
			s.Color = colors.Uniform(colors.FromRGB(0, 102, 255))
		})
		txtplayerScore := core.Bind(&strPlayerScoreLbl, core.NewText(frplayerScore))
		txtplayerScore.Styler(func(s *styles.Style) {
			s.Color = colors.Uniform(colors.FromRGB(0, 102, 255))
		})
		strPlayerScoreLbl = CUserName + ": R" + strconv.Itoa(playerScore)
		txtplayerScore.Update()

		frai1Score := core.NewFrame(&pg.Frame)
		frai1Score.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
			s.Min.Set(units.Pw(35), units.Ph(10))
			s.Max.Set(units.Pw(35), units.Ph(10))
			s.Pos.Set(units.Pw(55), units.Ph(50))
			s.RenderBox = false
			s.Background = colors.Uniform(colors.FromRGB(18, 19, 22))
		})
		ai1TokenID := core.NewIcon(frai1Score)
		ai1TokenID.Styler(func(s *styles.Style) {
			s.IconSize.Set(units.Vw(5))
			s.Color = colors.Uniform(colors.FromRGB(255, 47, 0))
		})
		txtai1Score := core.NewText(frai1Score)
		txtai1Score.Styler(func(s *styles.Style) {
			s.Color = colors.Uniform(colors.FromRGB(255, 47, 0))
		})
		txtai1Score.SetText(ais[Ai1].Name + ": R" + strconv.Itoa(intai1Score))

		frai2Score := core.NewFrame(&pg.Frame)
		frai2Score.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
			s.Min.Set(units.Pw(35), units.Ph(10))
			s.Max.Set(units.Pw(35), units.Ph(10))
			s.Pos.Set(units.Pw(10), units.Ph(65))
			s.RenderBox = false
			s.Background = colors.Uniform(colors.FromRGB(18, 19, 22))
		})
		ai2TokenID := core.NewIcon(frai2Score)
		ai2TokenID.Styler(func(s *styles.Style) {
			s.IconSize.Set(units.Vw(5))
			s.Color = colors.Uniform(colors.FromRGB(255, 47, 0))
		})
		txtai2Score := core.NewText(frai2Score)
		txtai2Score.Styler(func(s *styles.Style) {
			s.Color = colors.Uniform(colors.FromRGB(255, 47, 0))
		})
		txtai2Score.SetText(ais[Ai2].Name + ": R" + strconv.Itoa(intai2Score))

		frai3Score := core.NewFrame(&pg.Frame)
		frai3Score.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
			s.Min.Set(units.Pw(35), units.Ph(10))
			s.Max.Set(units.Pw(25), units.Ph(10))
			s.Pos.Set(units.Pw(55), units.Ph(65))
			s.RenderBox = false
			s.Background = colors.Uniform(colors.FromRGB(18, 19, 22))
		})
		ai3TokenID := core.NewIcon(frai3Score)
		ai3TokenID.Styler(func(s *styles.Style) {
			s.IconSize.Set(units.Vw(5))
			s.Color = colors.Uniform(colors.FromRGB(255, 47, 0))
		})
		txtai3Score := core.NewText(frai3Score)
		txtai3Score.Styler(func(s *styles.Style) {
			s.Color = colors.Uniform(colors.FromRGB(255, 47, 0))
		})
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
			s.Min.Set(units.Pw(20), units.Ph(10))
			s.Max.Set(units.Pw(20), units.Ph(10))
			s.Pos.Set(units.Pw(40), units.Ph(85))
			s.Background = colors.Uniform(colors.FromRGB(24, 4, 48))
			s.Color = colors.Uniform(colors.FromRGB(134, 0, 204))
		})

		btnExitGame.OnClick(func(e events.Event) {
			timeNow := time.Now()
			numNanosec := timeNow.Sub(startTime).Nanoseconds()
			CurrentUser.TotalTime += numNanosec
			CurrentUser.TotalScore += Session.Score
			err = MBSEGames.AddSession(ctx, firestoreClient, CurrentUser.DisplayName, Session.Subject, Session.Gamemode, Session.Ai1, Session.Ai2, Session.Ai3, Session.StartTime, timeNow.Format(time.RFC822), Session.QDetails, Session.Score, Session.CorrectAnswers, Session.IncorrectAnswers)
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

			err, _ = MBSEGames.EditUser(ctx, firestoreClient, CurrentUser.DocRef, "LastDate", "", 0, 0)
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
		XVal.Set(0, units.UnitPw)
		YVal.Set(0, units.UnitPw)

		var ai1Miss, ai2Miss, ai3Miss bool
		playerMiss = false
		ai1Miss = false
		ai2Miss = false
		ai3Miss = false

		playerToken := core.NewIcon(&pg.Frame)
		playerToken.Styler(func(s *styles.Style) {
			s.Pos.Set(units.Pw(2.5), units.Pw(87.5))
			s.IconSize.Set(units.Pw(4.1666666667))
			s.Color = colors.Uniform(colors.FromRGB(0, 102, 255))
			//s.Grow.Set(1, 1)
		})
		ai1Token := core.NewIcon(&pg.Frame)
		ai1Token.Styler(func(s *styles.Style) {
			s.Pos.Set(units.Pw(10), units.Pw(87.5))
			s.IconSize.Set(units.Pw(4.1666666667))
			s.Color = colors.Uniform(colors.FromRGB(255, 47, 0))
		})
		ai2Token := core.NewIcon(&pg.Frame)
		ai2Token.Styler(func(s *styles.Style) {
			s.Pos.Set(units.Pw(2.5), units.Pw(93.3333333333))
			s.IconSize.Set(units.Pw(4.1666666667))
			s.Color = colors.Uniform(colors.FromRGB(255, 47, 0))
		})
		ai3Token := core.NewIcon(&pg.Frame)
		ai3Token.Styler(func(s *styles.Style) {
			s.Pos.Set(units.Pw(10), units.Pw(93.3333333333))
			s.IconSize.Set(units.Pw(4.1666666667))
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
			s.Min.Set(units.Pw(20), units.Ph(10))
			s.Max.Set(units.Pw(20), units.Ph(10))
			s.Pos.Set(units.Pw(40), units.Ph(75))
			s.Background = colors.Uniform(colors.FromRGB(24, 4, 48))
			s.Color = colors.Uniform(colors.FromRGB(134, 0, 204))
		})
		btnRoll.OnClick(func(e events.Event) {
			if Session.StartTime == "" {
				Session.StartTime = time.Now().Format(time.RFC822)
			}

			r := *rand.New(rand.NewSource(time.Now().Unix()))
			//var boolCorrect bool
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
						err = MBSEGames.AddSession(ctx, firestoreClient, CurrentUser.DisplayName, Session.Subject, Session.Gamemode, Session.Ai1, Session.Ai2, Session.Ai3, Session.StartTime, timeNow.Format(time.RFC822), Session.QDetails, Session.Score, Session.CorrectAnswers, Session.IncorrectAnswers)
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
						err = MBSEGames.AddSession(ctx, firestoreClient, CurrentUser.DisplayName, Session.Subject, Session.Gamemode, Session.Ai1, Session.Ai2, Session.Ai3, Session.StartTime, timeNow.Format(time.RFC822), Session.QDetails, Session.Score, Session.CorrectAnswers, Session.IncorrectAnswers)
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
						err = MBSEGames.AddSession(ctx, firestoreClient, CurrentUser.DisplayName, Session.Subject, Session.Gamemode, Session.Ai1, Session.Ai2, Session.Ai3, Session.StartTime, timeNow.Format(time.RFC822), Session.QDetails, Session.Score, Session.CorrectAnswers, Session.IncorrectAnswers)
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
}

func Move(intCurrentPos, intRoll, tokenNum int, bd *core.Frame, token *core.Icon) (intScoreGained int) {
	intScoreGained = 0
	var XVal, YVal units.Value
	XVal.Set(0, units.UnitVw)
	YVal.Set(0, units.UnitVw)
	
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
	XVal.Set(0, units.UnitVw)
	YVal.Set(0, units.UnitVw)
	
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
			case 1:
				strAnsGive = "1"
			case 2:
				strAnsGive = "2"
			case 3:
				strAnsGive = "3"
			case 4:
				strAnsGive = "4"
			default:
				strAnsGive = "0"
			}
			if intAns == selVal {
				Session.CorrectAnswers++
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
				Session.IncorrectAnswers++
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

func restLogin(email, password string) (string, string, error) {
	url := "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=" + MBSEGames.WebApiKey
	payload := map[string]interface{}{
		"email":             email,
		"password":          password,
		"returnSecureToken": true,
	}
	jsonBody, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("login failed: %d", resp.StatusCode)
	}

	var result authResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}
	return result.IdToken, result.Email, nil
}

// --- Post-Login Handler ---

func handleLoginSuccess(token string, email string) {
	log.Println("-----------------------------------------")
	log.Println("Login Successful!")
	log.Printf("ID Token received: %s...\n", token[:15])

	currentUseremail = email

	log.Printf("User Logged In: %s\n", email)
	log.Println("-----------------------------------------")

	// You can now use 'firestoreClient' to fetch data!
	// Example: fetchUserData(context.Background())
}
