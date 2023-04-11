package secret

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
)

type hasher struct {
	hashFunc  hashFunc
	validFunc compareFunc
}

var (
	day0Hasher hasher = hasher{hash0SHA256, compare0SHA256}
	day1Hasher hasher = hasher{hash1MD5, compare1MD5}
	day2Hasher hasher = hasher{hash2SHA512, compare2SHA512}
	day3Hasher hasher = hasher{hash3SHA384, compare3SHA384}
	day4Hasher hasher = hasher{hash4Bcrypt, compare4Bcrypt}
	day5Hasher hasher = hasher{hash5ARGON2, compare5ARGON2}
	day6Hasher hasher = hasher{hash6PBKDF2, compare6PBKDF2}
)

var weekDays map[string]hasher = map[string]hasher{
	"0-S$": day0Hasher,
	"1-M$": day1Hasher,
	"2-T$": day2Hasher,
	"3-W$": day3Hasher,
	"4-T$": day4Hasher,
	"5-F$": day5Hasher,
	"6-S$": day6Hasher,
}

type hashFunc func(data, salt []byte) string

type compareFunc func(data, salt []byte, hash string) bool

// Recommended use is only in testing!
func HashForDay(data, salt []byte, day time.Weekday) string {
	var b strings.Builder
	var dayString = strconv.Itoa(int(day))
	var weekDayString = day.String()[0:1]
	b.WriteString(dayString)
	b.WriteString("-")
	b.WriteString(weekDayString)
	b.WriteString("$")
	var hash, ok = weekDays[b.String()]
	if !ok {
		panic("Hasher not found")
	}
	b.WriteString(hash.hashFunc(data, salt))
	return b.String()
}

// Hashes the data with the salt, if the algorithm for that day supports it.
// If not, hash the data, appending the salt to it.
func Hash(data, salt []byte) string {
	var day = time.Now().Weekday()
	return HashForDay(data, salt, day)
}

// Compares the data with the salt and the hash.
// If the hash is not in the correct format, it will return false.
func Compare(data, salt []byte, hash string) bool {
	if len(hash) <= 4 {
		return false
	}
	var hashNum = hash[0:4]
	var hasher, ok = weekDays[hashNum]
	if !ok {
		return false
	}
	hash = hash[4:]
	return hasher.validFunc(data, salt, hash)
}

func hash0SHA256(data, salt []byte) string {
	var hash = sha256.Sum256(bytes.Join([][]byte{data, salt}, []byte{}))
	return hex.EncodeToString(hash[:])
}

func compare0SHA256(data, salt []byte, hash string) bool {
	return hash0SHA256(data, salt) == hash
}

func hash1MD5(data, salt []byte) string {
	var hash = md5.Sum(bytes.Join([][]byte{data, salt}, []byte{}))
	return hex.EncodeToString(hash[:])
}

func compare1MD5(data, salt []byte, hash string) bool {
	return hash1MD5(data, salt) == hash
}

func hash2SHA512(data, salt []byte) string {
	var hash = sha512.Sum512(bytes.Join([][]byte{data, salt}, []byte{}))
	return hex.EncodeToString(hash[:])
}

func compare2SHA512(data, salt []byte, hash string) bool {
	return hash2SHA512(data, salt) == hash
}

func hash3SHA384(data, salt []byte) string {
	var hash = sha512.Sum384(bytes.Join([][]byte{data, salt}, []byte{}))
	return hex.EncodeToString(hash[:])
}

func compare3SHA384(data, salt []byte, hash string) bool {
	return hash3SHA384(data, salt) == hash
}

func hash4Bcrypt(data, salt []byte) string {
	bcryptHash, _ := bcrypt.GenerateFromPassword(bytes.Join([][]byte{data, salt}, []byte{}), bcrypt.DefaultCost)
	return string(bcryptHash)
}

func compare4Bcrypt(data, salt []byte, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), bytes.Join([][]byte{data, salt}, []byte{})) == nil
}

func hash5ARGON2(data, salt []byte) string {
	var hash = argon2.Key(data, salt, 1, 64*1024, 4, 32)
	return hex.EncodeToString(hash)
}

func compare5ARGON2(data, salt []byte, hash string) bool {
	return hash5ARGON2(data, salt) == hash
}

func hash6PBKDF2(data, salt []byte) string {
	var hash = pbkdf2.Key(data, salt, 100000, 32, sha256.New)
	return hex.EncodeToString(hash)
}

func compare6PBKDF2(data, salt []byte, hash string) bool {
	return hash6PBKDF2(data, salt) == hash
}
