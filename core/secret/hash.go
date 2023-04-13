package secret

import (
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

// This package provides a way to generate a different type of hash for each day of the week.
//
// If the hashing algorithm does not allow for a salt, we will append the salt to the data.
//
// The hashing algorithm is chosen based on the day of the week.
//
// Algorithms:
// 0 - SHA256 (Sunday)
// 1 - MD5 (Monday)
// 2 - SHA512 (Tuesday)
// 3 - SHA384 (Wednesday)
// 4 - Bcrypt (Thursday)
// 5 - ARGON2 (Friday)
// 6 - PBKDF2 (Saturday)
//
// The hash is in the format: <day>-<first letter of day>$<hash>
// Example: 0-S$<hash>
//
// Multiple iterations of hashing is not supported.
// IE: If the hash is 0-S$<hash>, it should not be hashed again.
//
// We allow overwriting of the daily hashers.

var (
	Day0Hasher = Hasher{hash0SHA256, compare0SHA256} // Sunday     (SHA256)
	Day1Hasher = Hasher{hash1MD5, compare1MD5}       // Monday     (MD5)
	Day2Hasher = Hasher{hash2SHA512, compare2SHA512} // Tuesday    (SHA512)
	Day3Hasher = Hasher{hash3SHA384, compare3SHA384} // Wednesday  (SHA384)
	Day4Hasher = Hasher{hash4Bcrypt, compare4Bcrypt} // Thursday   (Bcrypt)
	Day5Hasher = Hasher{hash5ARGON2, compare5ARGON2} // Friday     (ARGON2)
	Day6Hasher = Hasher{hash6PBKDF2, compare6PBKDF2} // Saturday   (PBKDF2)
)

var weekDays = map[string]Hasher{
	"0-S$": Day0Hasher, // Sunday
	"1-M$": Day1Hasher, // Monday
	"2-T$": Day2Hasher, // Tuesday
	"3-W$": Day3Hasher, // Wednesday
	"4-T$": Day4Hasher, // Thursday
	"5-F$": Day5Hasher, // Friday
	"6-S$": Day6Hasher, // Saturday
}

// A hash function should return a hash of the data and salt.
type HashFunc func(data, salt []byte) string

// A compare function should return true if the data and salt match the hash.
type CompareFunc func(data, salt []byte, hash string) bool

// A hasher is a struct that contains a hash function and a compare function.
//
// The hash function should return a hash of the data and salt.
//
// The compare function should return true if the data and salt match the hash.
type Hasher struct {
	hashFunc  HashFunc
	validFunc CompareFunc
}

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
	var hash = sha256.Sum256(append(data, salt...))
	return hex.EncodeToString(hash[:])
}

func compare0SHA256(data, salt []byte, hash string) bool {
	return hash0SHA256(data, salt) == hash
}

func hash1MD5(data, salt []byte) string {
	var hash = md5.Sum(append(data, salt...))
	return hex.EncodeToString(hash[:])
}

func compare1MD5(data, salt []byte, hash string) bool {
	return hash1MD5(data, salt) == hash
}

func hash2SHA512(data, salt []byte) string {
	var hash = sha512.Sum512(append(data, salt...))
	return hex.EncodeToString(hash[:])
}

func compare2SHA512(data, salt []byte, hash string) bool {
	return hash2SHA512(data, salt) == hash
}

func hash3SHA384(data, salt []byte) string {
	var hash = sha512.Sum384(append(data, salt...))
	return hex.EncodeToString(hash[:])
}

func compare3SHA384(data, salt []byte, hash string) bool {
	return hash3SHA384(data, salt) == hash
}

func hash4Bcrypt(data, salt []byte) string {
	bcryptHash, _ := bcrypt.GenerateFromPassword(append(data, salt...), bcrypt.DefaultCost)
	return string(bcryptHash)
}

func compare4Bcrypt(data, salt []byte, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), append(data, salt...)) == nil
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
