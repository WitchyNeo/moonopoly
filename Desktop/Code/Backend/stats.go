package MBSEGames

import (
	"strconv"
	"time"

	MBSEGames "MBSEGames/Code/Backend/Database"
)

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//Structures & Variables
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
type StatsSummary struct {
	AvgCSession float64
	AvgISession float64
	AvgScSession float64
	AvgMinSession float64
	AvgASession float64
	AvgQperMin float64
	AvgTotCPlayers float64
	AvgTotIPlayers float64
	AvgTotScPlayers float64
	AvgTotMinPlayers float64
	AvgAPlayers float64
	AvgImpPlayers float64
}

type QuestionStats struct {
	QRef string
	TotAsked int
	ModeAns int
	TotCorrect int
	TotIncorrect int
	TotAccuracy float64
	Num1 int
	Num2 int
	Num3 int
	Num4 int
}

type SesQData struct {
	QRef string
	AnsGiven int
	IsCorrect bool
}

type SessionStats struct {
	SRef string
	PlayerDN string
	TimePlayed int
	Accuracy float64
	AvgQperMin float64
	QStats []SesQData
}

type PlayerStats struct {
	PRef string
	PlayerDN string
	AvgMinSession float64
	AvgCSession float64
	AvgISession float64
	AvgASession float64
	AvgScSession float64
	AvgImprovement float64
	Imps []Improvement
}

type PlayerStatCalc struct {
	PRef string
	SCount int
	CCount int
	ICount int
	TotSc  int
	TotMin int
	TotAcc float64
}

type Improvement struct {
	Ses1Ref string
	Ses2Ref string
	PercentDiffQpM float64
	PercentDiffAcc float64
	TotImp float64
}

var RawSessions []MBSEGames.Session
var StatSum StatsSummary
var SStats []SessionStats
var QStats []QuestionStats
var Pstats []PlayerStats
var pStatCalc []PlayerStatCalc
var totCorrect, totIncorrect, numSessions, numPlayers, totMin, totScore int
var sumAccuracy, sumAccPlayers, sumQpM, sumImp float64

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//Sessions
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//main stat processor
func CalcStats() {
	processPlayers(MBSEGames.Users)
	for _, session := range RawSessions {
		var ss SessionStats
		ss.SRef = session.DocRef
		for i, p := range Pstats {
			if p.PlayerDN == session.UserDisplayName {
				var tempAcc float64
				pStatCalc[i].SCount++
				pStatCalc[i].CCount += session.NumCorrect
				pStatCalc[i].ICount += session.NumIncorrect
				startT, _ := time.Parse(time.RFC3339, session.StartTime)
				endT, _ := time.Parse(time.RFC3339, session.EndTime)
				sesLenMin := int(endT.Sub(startT).Minutes())
				pStatCalc[i].TotMin += sesLenMin
				tempQpM := float64((pStatCalc[i].CCount + pStatCalc[i].ICount) / sesLenMin)
				sumQpM += tempQpM
				sesQs := session.NumCorrect + session.NumIncorrect
				tempAcc = float64(session.NumCorrect / sesQs)
				pStatCalc[i].TotAcc += tempAcc
				pStatCalc[i].TotSc += int(session.Score)
				Pstats[i].AvgASession = float64(int(pStatCalc[i].TotAcc) / pStatCalc[i].SCount)
				Pstats[i].AvgCSession = float64(pStatCalc[i].CCount / pStatCalc[i].SCount)
				Pstats[i].AvgISession = float64(pStatCalc[i].ICount / pStatCalc[i].SCount)
				Pstats[i].AvgMinSession = float64(pStatCalc[i].TotMin / pStatCalc[i].SCount)
				Pstats[i].AvgScSession = float64(pStatCalc[i].TotSc / pStatCalc[i].SCount)
				ss.Accuracy = tempAcc
				ss.PlayerDN = session.UserDisplayName
				ss.TimePlayed = sesLenMin
				ss.AvgQperMin = tempQpM
				sumAccPlayers += tempAcc
				totMin += sesLenMin
			}
		}
		ss.QStats = processQuestions(session)
		SStats = append(SStats, ss)
		totCorrect += session.NumCorrect
		totIncorrect += session.NumIncorrect
		totScore += int(session.Score)
		numSessions++
		sumAccuracy += float64(session.NumCorrect / (session.NumCorrect + session.NumIncorrect))

		StatSum.AvgCSession = float64(totCorrect / numSessions)
		StatSum.AvgISession = float64(totIncorrect / numSessions)
		StatSum.AvgScSession = float64(totScore / numSessions)
		StatSum.AvgMinSession = float64(totMin / numSessions)
		StatSum.AvgASession = sumAccuracy / float64(numSessions)
		StatSum.AvgQperMin = sumQpM / float64(numSessions)
		StatSum.AvgTotCPlayers = float64(totCorrect / numPlayers)
		StatSum.AvgTotIPlayers = float64(totIncorrect / numPlayers)
		StatSum.AvgTotScPlayers = float64(totScore / numPlayers)
		StatSum.AvgTotMinPlayers = float64(totMin / numPlayers)
		StatSum.AvgAPlayers = sumAccPlayers / float64(numPlayers)
		calcImprovement()
	}
}


///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//Players
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//Register all players into proper struct
func processPlayers(rawPlayers []MBSEGames.User) {
	for _, player := range rawPlayers {
		var p PlayerStats
		var pc PlayerStatCalc
		p.PRef = player.DocRef
		p.PlayerDN = player.DisplayName
		p.AvgScSession = 0
		p.AvgASession = 0
		p.AvgCSession = 0
		p.AvgISession = 0
		p.AvgImprovement = 0
		p.AvgMinSession = 0

		pc.PRef = player.DocRef
		pc.CCount = 0
		pc.ICount = 0
		pc.SCount = 0
		pc.TotSc = 0
		pc.TotMin = 0
		pc.TotAcc = 0

		Pstats = append(Pstats, p)
		pStatCalc = append(pStatCalc, pc)
		numPlayers++
	}
}

func calcImprovement() {
	for i := range Pstats {
		var prevS SessionStats
		pName := Pstats[i].PlayerDN
		for j, s := range SStats {
			if pName == s.PlayerDN {
				if j == 0 {
					prevS = s
				} else {
					var imp Improvement
					imp.Ses1Ref = prevS.SRef
					imp.Ses2Ref = s.SRef
					imp.PercentDiffQpM = (s.AvgQperMin / prevS.AvgQperMin) - 1
					imp.PercentDiffAcc = s.Accuracy - prevS.Accuracy
					imp.TotImp = imp.PercentDiffAcc + imp.PercentDiffQpM
					Pstats[i].Imps = append(Pstats[i].Imps, imp)
					prevS = s
					sumImp += imp.TotImp
				}
			}
		}
	}
	StatSum.AvgImpPlayers = sumImp / float64(numPlayers)
}


///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//Questions
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//Split & Stat Qs
func processQuestions(rawSession MBSEGames.Session) []SesQData {
	for _, question := range MBSEGames.Questions {
		var qFinStats QuestionStats
		qFinStats.QRef = question.DocRef
		qFinStats.TotAsked = 0
		qFinStats.TotCorrect = 0
		qFinStats.TotIncorrect = 0
		qFinStats.TotAccuracy = 0
		qFinStats.ModeAns = 0
		qFinStats.Num1 = 0
		qFinStats.Num2 = 0
		qFinStats.Num3 = 0
		qFinStats.Num4 = 0

		QStats = append(QStats, qFinStats)
	}
	var qStats []SesQData
	numQuestions := rawSession.NumCorrect + rawSession.NumIncorrect
	for i := 0; i < numQuestions; i++ {
		var qs SesQData
		qdata := rawSession.QDetails[25*i:25]
		qref := qdata[:20]
		ans, _ := strconv.Atoi(qdata[20:21])
		var iscorrect bool
		if qdata[24:25] == "0" {
			iscorrect = false
		} else {
			iscorrect = true
		}
		qs.QRef = qref
		qs.AnsGiven = ans
		qs.IsCorrect = iscorrect
		qStats = append(qStats, qs)

		for j, qfinstat := range QStats {
			if qref == qfinstat.QRef {
				QStats[j].TotAsked++
				if qs.IsCorrect {
					QStats[j].TotCorrect++
				} else {
					QStats[j].TotIncorrect++
				}

				switch qs.AnsGiven {
				case 1: QStats[j].Num1++
				case 2: QStats[j].Num2++
				case 3: QStats[j].Num3++
				case 4: QStats[j].Num4++
				}

				if QStats[j].Num1 > QStats[j].Num2 && QStats[j].Num1 > QStats[j].Num3 && QStats[j].Num1 > QStats[j].Num4 {
					QStats[j].ModeAns = 1
				} else if QStats[j].Num2 > QStats[j].Num1 && QStats[j].Num2 > QStats[j].Num3 && QStats[j].Num2 > QStats[j].Num4 {
					QStats[j].ModeAns = 2
				} else if QStats[j].Num3 > QStats[j].Num2 && QStats[j].Num3 > QStats[j].Num1 && QStats[j].Num3 > QStats[j].Num4 {
					QStats[j].ModeAns = 3
				} else if QStats[j].Num4 > QStats[j].Num2 && QStats[j].Num4 > QStats[j].Num3 && QStats[j].Num4 > QStats[j].Num1{
					QStats[j].ModeAns = 4
				} else {
					QStats[j].ModeAns = 0
				}

				QStats[j].TotAccuracy = float64(QStats[j].TotCorrect / (QStats[j].TotCorrect + QStats[j].TotIncorrect))
			}
		} 
	}
	
	return qStats
}
