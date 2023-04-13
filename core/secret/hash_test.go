package secret_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/core/secret"
)

func TestHash(t *testing.T) {
	var mapToHash = map[int][]byte{
		0: []byte( /* bytes to hash (Sunday)	*/ "hash-me" + time.Now().Add(time.Hour*3).String()),
		1: []byte( /* bytes to hash (Monday)	*/ "hash-me" + time.Now().Add(time.Hour*6).String()),
		2: []byte( /* bytes to hash (Tuesday)	*/ "hash-me" + time.Now().Add(time.Hour*12).String()),
		3: []byte( /* bytes to hash (Wednesday)	*/ "hash-me" + time.Now().Add(time.Hour*24).String()),
		4: []byte( /* bytes to hash (Thursday)	*/ "hash-me" + time.Now().Add(time.Hour*48).String()),
		5: []byte( /* bytes to hash (Friday)	*/ "hash-me" + time.Now().Add(time.Hour*96).String()),
		6: []byte( /* bytes to hash (Saturday)	*/ "hash-me" + time.Now().Add(time.Hour*192).String()),
	}
	var salt = []byte("salt")
	for k, v := range mapToHash {
		var day = time.Weekday(k)
		var hash = secret.HashForDay(v, salt, day)
		var dayString = strconv.Itoa(int(day))
		var weekDayString = day.String()[0:1]
		var hashNum = fmt.Sprintf("%s-%s$", dayString, weekDayString)
		if !strings.HasPrefix(hash, hashNum) {
			t.Errorf("Hashing failed for day %d, %s %s %s", k, hash, hashNum, dayString)
		}

		if cmp := secret.Compare(v, salt, hash); !cmp {
			t.Errorf("Hashing failed for day %d, %s %s %s", k, hash, hashNum, dayString)
		} else {
			t.Log("Hashing passed for", day.String(), hash)
		}
	}
}
