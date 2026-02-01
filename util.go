package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/DSU-DefSec/DWAYNE-INATOR-5000/checks"
	"github.com/gin-gonic/gin"
)

func errorOutGraceful(c *gin.Context, err error) {
	errorPrint("error:", err)
	c.Redirect(http.StatusSeeOther, "/")
	c.Abort()
}

func errorOutAnnoying(c *gin.Context, err error) {
	errorPrint("error:", err)
	c.Redirect(http.StatusSeeOther, "/forbidden")
	c.Abort()
}

func addDelayedChecks() {
	// go in reverse to make truncation easier
	for i := len(delayedChecks.Box) - 1; i >= 0; i-- {
		if time.Now().After(delayedChecks.Box[i].InjectTime()) {

			delayedBox := delayedChecks.Box[i]

			// remove box from list
			delayedChecks.Box[i] = delayedChecks.Box[len(delayedChecks.Box)-1]
			delayedChecks.Box = delayedChecks.Box[:len(delayedChecks.Box)-1]

			boxList := []Box{delayedBox}
			err := validateChecks(boxList)
			if err != nil {
				log.Println("[ERROR] Check validation on delayed check:", delayedBox.Name, err)
				continue
			}

			delayedBox = boxList[0]

			boxIndex := -1
			for j, b := range dwConf.Box {
				if b.Name == delayedBox.Name {
					boxIndex = j
				}
			}

			if boxIndex < 0 {
				// Add new box
				dwConf.Box = append(dwConf.Box, delayedBox)
			} else {
				// Add new checks
				for _, c := range delayedBox.CheckList {
					dwConf.Box[boxIndex].CheckList = append(dwConf.Box[boxIndex].CheckList, c)
				}
			}

		}
	}
}

func makeResultsMap(resList []ResultEntry) map[string]ResultEntry {
	resMap := make(map[string]ResultEntry)
	for _, r := range resList {
		resMap[r.Name] = r
	}
	return resMap
}

func validateString(input string) bool {
	if input == "" {
		return false
	}
	validationString := `^[a-zA-Z0-9-_]+$`
	inputValidation := regexp.MustCompile(validationString)
	return inputValidation.MatchString(input)
}

func (t TeamData) IsValid() bool {
	return t.Name != ""
}

func (m *config) getCheck(checkName string) (checks.Check, error) {
	for _, box := range m.Box {
		for _, check := range box.CheckList {
			if check.FetchName() == checkName {
				return check, nil
			}
		}
	}
	return checks.Web{}, errors.New("check not found")
}

func calculateScoreTotal(rec TeamRecord) int {
	total := (rec.ServicePoints * dwConf.ServicePoints) + rec.InjectPoints
	total -= rec.RedTeamPoints + (rec.SlaViolations * dwConf.SlaPoints)
	if dwConf.Persists {
		total += rec.PointsStolen + rec.PersistPoints
		total -= rec.PointsLost
	}
	total += rec.ManualAdjustment
	return total
}

func readInject(inj InjectSubmission) string {
	content, err := os.ReadFile("submissions/" + inj.DiskFile)
	if err != nil {
		errorPrint(err)
		return ""
	}
	return string(content)
}

func boxFromIP(ip string) (TeamData, string, error) {
	for _, box := range dwConf.Box {
		for _, t := range dwConf.Team {
			if ip == strings.Replace(box.IP, "x", t.IP, 1) {
				return t, box.Name, nil
			}
		}
	}
	return TeamData{}, "", errors.New("box not found")
}

func tokenToTeam(token string) (TeamData, error) {
	for _, t := range dwConf.Team {
		if t.Token == token {
			return t, nil
		}
	}
	return TeamData{}, errors.New("invalid token")
}

func (m *config) GetTeam(teamID uint) (TeamData, error) {
	for _, team := range m.Team {
		if team.ID == teamID {
			return team, nil
		}
	}
	return TeamData{}, errors.New("team not found")
}

func oneOfN(points, parties int) int {
	return int(float64(points)/float64(parties) + 0.5)
}

func resetEvent(c *gin.Context) {
	team := getUser(c)
	if !team.IsAdmin() {
		errorOutAnnoying(c, errors.New("non-admin tried to issue a scoring engine reset: "+c.Param("team")))
		return
	}

	teamMutex.Lock()
	resetIssued = true

	db.Exec("DELETE FROM result_entries")
	db.Exec("DELETE FROM team_records")
	db.Exec("DELETE FROM inject_submissions")
	db.Exec("DELETE FROM slas")
	db.Exec("DELETE FROM persists")

	// Deal with cache
	cachedStatus = []TeamRecord{}
	cachedRound = 0
	roundNumber = 0
	startTime = time.Now().In(loc)
	persistHits = make(map[uint]map[string][]uint)
	teamMutex.Unlock()

	c.Redirect(http.StatusSeeOther, "/")
}

func pauseEvent(c *gin.Context) {
	team := getUser(c)
	if !team.IsAdmin() {
		errorOutAnnoying(c, errors.New("non-admin tried to start scoring: "+c.Param("team")))
		return
	}
	dwConf.Running = false
	resetIssued = true
	pauseTime = time.Now()
	c.Redirect(http.StatusSeeOther, "/settings")
}
