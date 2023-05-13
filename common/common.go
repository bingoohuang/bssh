// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

// Package common is a package that summarizes the common processing of bssh package.

package common

import (
	"bufio"
	"bytes"
	"crypto/sha1" // nolint
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli"
	"golang.org/x/term"
)

// nolint
var characterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// IsExist returns existence of file.
func IsExist(filename string) (string, bool) {
	fi, err := os.Lstat(filename)
	if err != nil {
		log.Printf("stat %s: %v", filename, err)
		return filename, false
	}

	if fi.Mode()&os.ModeSymlink != 0 {
		s, err := os.Readlink(filename)
		if err != nil {
			log.Printf("readlink %s: %v", filename, err)
			return filename, false
		}
		return s, true
	}

	return filename, true
}

// MapReduce sets map1 value to map2 if map1 and map2 have same key, and value
// of map2 is zero value. Available interface type is string or []string or
// bool.
//
// WARN: This function returns a map, but updates value of map2 argument too.
func MapReduce(map1, map2 map[string]interface{}) map[string]interface{} {
	for ia, va := range map1 {
		switch value := va.(type) {
		case string:
			if value != "" && map2[ia] == "" {
				map2[ia] = value
			}
		case []string:
			map2Value := reflect.ValueOf(map2[ia])
			if len(value) > 0 && map2Value.Len() == 0 {
				map2[ia] = value
			}
		case bool:
			map2Value := reflect.ValueOf(map2[ia])
			if value && !map2Value.Bool() {
				map2[ia] = value
			}
		}
	}

	return map2
}

// StructToMap returns a map that converted struct to map.
// Keys of map are set from public field of struct.
//
// WARN: ok value is not used. Always returns false.
func StructToMap(val interface{}) (mapVal map[string]interface{}, ok bool) {
	structVal := reflect.Indirect(reflect.ValueOf(val))
	typ := structVal.Type()

	mapVal = make(map[string]interface{})

	for i := 0; i < typ.NumField(); i++ {
		field := structVal.Field(i)

		if field.CanSet() {
			mapVal[typ.Field(i).Name] = field.Interface()
		}
	}

	return
}

// MapToStruct sets value of mapVal to public field of val struct.
// Raises panic if mapVal has keys of private field of val struct or field that
// val struct doesn't have.
//
// WARN: ok value is not used. Always returns false.
func MapToStruct(mapVal map[string]interface{}, val interface{}) (ok bool) {
	structVal := reflect.Indirect(reflect.ValueOf(val))

	for name, elem := range mapVal {
		structVal.FieldByName(name).Set(reflect.ValueOf(elem))
	}

	return
}

// GetFullPath returns a fullpath of path.
// Expands `~` to user directory ($HOME environment variable).
func GetFullPath(path string) (fullPath string) {
	fullPath = ExpandHomeDir(path)
	fullPath, _ = filepath.Abs(fullPath)

	return fullPath
}

// GetOrderNumber get order num in array.
func GetOrderNumber(value string, array []string) int {
	for i, v := range array {
		if v == value {
			return i
		}
	}

	return 0
}

// GetMaxLength returns a max byte length of list.
func GetMaxLength(list []string) int {
	maxLength := 0
	for _, elem := range list {
		if maxLength < len(elem) {
			maxLength = len(elem)
		}
	}

	return maxLength
}

// GetFilesBase64 returns a base64 encoded string of file content of paths.
func GetFilesBase64(paths []string) (string, error) {
	var data bytes.Buffer

	for _, path := range paths {
		file, err := os.ReadFile(GetFullPath(path))
		if err != nil {
			return "", err
		}

		data.Write(file)
		data.WriteByte('\n')
	}

	result := base64.StdEncoding.EncodeToString(data.Bytes())

	return result, nil
}

// GetPassPhrase gets the passphrase from virtual terminal input and returns the result. Works only on UNIX-based OS.
func GetPassPhrase(msg string) (input string, err error) {
	fmt.Print(msg)

	// Open /dev/tty
	tty, err := os.Open("/dev/tty")
	if err != nil {
		log.Fatal(err)
	}
	defer tty.Close()

	// get input
	result, err := term.ReadPassword(int(tty.Fd()))

	if len(result) == 0 {
		err = fmt.Errorf("err: input is empty")
		return
	}

	input = string(result)

	fmt.Println()

	return
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// NewSHA1Hash generates a new SHA1 hash based on
// a random number of characters.
func NewSHA1Hash(n ...int) string {
	noRandomCharacters := 32

	if len(n) > 0 {
		noRandomCharacters = n[0]
	}

	randString := RandomString(noRandomCharacters)

	hash := sha1.New() // nolint
	_, _ = hash.Write([]byte(randString))
	bs := hash.Sum(nil)

	return fmt.Sprintf("%02x", bs)
}

// RandomString generates a random string of n length.
func RandomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = characterRunes[rand.Intn(len(characterRunes))]
	}

	return string(b)
}

// GetUniqueSlice return slice, removes duplicate values ​​from data(slice).
func GetUniqueSlice(data []string) (result []string) {
	m := make(map[string]bool)

	for _, ele := range data {
		if !m[ele] {
			m[ele] = true

			result = append(result, ele)
		}
	}

	return
}

// WalkDir return file path list ([]string).
func WalkDir(dir string) (files []string, err error) {
	_, err = os.Lstat(dir)
	if err != nil {
		return
	}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if IsHidden(dir, path) {
			return nil // ignore hidden filesß
		}

		if info.IsDir() {
			path += "/"
		}

		files = append(files, path)

		return nil
	})

	return
}

// IsHidden tells a path is hidden after basedir.
func IsHidden(basedir, path string) bool {
	p, err := filepath.Rel(basedir, path)
	if err != nil {
		p = path
	}

	parts := filepath.SplitList(p)

	for _, part := range parts {
		if len(part) > 1 && part[0:1] == "." {
			return true
		}
	}

	return false
}

// GetIDFromName return user name from /etc/passwd and uid.
func GetIDFromName(file string, name string) (id uint32, err error) {
	rd := strings.NewReader(file)
	sc := bufio.NewScanner(rd)

	for sc.Scan() {
		l := sc.Text()
		line := strings.Split(l, ":")

		if line[0] == name {
			idstr := line[2]
			u64, _ := strconv.ParseUint(idstr, 10, 32)
			id = uint32(u64)

			return
		}
	}

	err = fmt.Errorf("error: %s", "name not found")

	return
}

// GetNameFromID return user name from /etc/passwd and uid.
func GetNameFromID(file string, id uint32) (name string, err error) {
	rd := strings.NewReader(file)
	sc := bufio.NewScanner(rd)

	idstr := strconv.FormatUint(uint64(id), 10)

	for sc.Scan() {
		l := sc.Text()
		line := strings.Split(l, ":")

		if line[2] == idstr {
			name = line[0]
			return
		}
	}

	err = fmt.Errorf("error: %s", "name not found")

	return
}

// ParseForwardPort return forward address and port from string.
//
// ex.)
//   - `localhost:8000:localhost:18000` => local: "localhost:8000", remote: "localhost:18000"
//   - `8080:localhost:18080` => local: "localhost:8080", remote: "localhost:18080"
//   - `localhost:2222:12222` => local: "localhost:2222", remote: "localhost:12222"
func ParseForwardPort(value string) (local, remote string, err error) {
	// count column
	count := strings.Count(value, ":")
	data := strings.Split(value, ":")

	// switch count
	switch count {
	case 3:
		// `localhost:8000:localhost:18000`
		local = data[0] + ":" + data[1]
		remote = data[2] + ":" + data[3]
	case 2:
		// check 1st column is int
		_, e := strconv.Atoi(data[0])
		if e == nil { // 1st column is port (int)
			local = "localhost:" + data[0]
			remote = data[1] + ":" + data[2]
		} else { // 1st column is not port (int)
			local = data[0] + ":" + data[1]
			remote = "localhost:" + data[2]
		}

	default:
		err = errors.New("could not parse")
	}

	return
}

var (
	optionReg = regexp.MustCompile("^-")
	parseReg  = regexp.MustCompile("^-[^-]{2,}")
)

// ParseArgs return os.Args parse short options (ex.) [-la] => [-l,-a] )
func ParseArgs(options []cli.Flag, args []string) []string {
	// create cli.Flag map
	optionMap := map[string]cli.Flag{}

	for _, op := range options {
		names := strings.Split(op.GetName(), ",")

		for _, n := range names {
			// add hyphen
			if len(n) == 1 {
				optionMap["-"+n] = op
			} else {
				optionMap["--"+n] = op
			}
		}
	}

	result := []string{args[0]}

	// command flag
	isOptionArgs := false

parseloop:
	for i, arg := range args[1:] {
		switch {
		case !optionReg.MatchString(arg) && !isOptionArgs:
			// not option arg, and sOptinArgs flag false
			result = append(result, args[i+1:]...)
			break parseloop

		case !optionReg.MatchString(arg) && isOptionArgs:
			result = append(result, arg)

		case parseReg.MatchString(arg): // combine short option -la)
			slice := strings.Split(arg[1:], "")
			for _, s := range slice {
				s = "-" + s
				result = append(result, s)

				if val, ok := optionMap[s]; ok {
					switch val.(type) {
					case cli.StringSliceFlag, cli.StringFlag:
						isOptionArgs = true
					}
				}
			}

		default: // options (-a,--all)
			result = append(result, arg)

			if val, ok := optionMap[arg]; ok {
				switch val.(type) {
				case cli.StringSliceFlag, cli.StringFlag:
					isOptionArgs = true
				}
			}
		}
	}

	return result
}
