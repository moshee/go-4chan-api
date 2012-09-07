package api

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"time"
	"net/http"
)

// Direct mapping from the API's JSON to a Go type. Only use if you really want to.
type JsonPost struct {
	No          int64  `json:"no"`           // Post number         1-9999999999999
	Resto       int64  `json:"resto"`        // Reply to            0 (is thread), 1-999999999999
	Sticky      int    `json:"sticky"`       // Stickied thread?    0 (no), 1 (yes)
	Closed      int    `json:"closed"`       // Closed thread?      0 (no), 1 (yes)
	Now         string `json:"now"`          // Date and time       MM\/DD\/YY(Day)HH:MM (:SS on some boards)
	Time        int64  `json:"time"`         // UNIX timestamp      UNIX timestamp
	Name        string `json:"name"`         // Name                text or empty
	Trip        string `json:"trip"`         // Tripcode            text (format: !tripcode!!securetripcode)
	Id          string `json:"id"`           // ID                  text (8 characters), Mod, Admin
	Capcode     string `json:"capcode"`      // Capcode             none, mod, admin, admin_highlight, developer
	Country     string `json:"country"`      // Country code        ISO 3166-1 alpha-2, XX (unknown)
	CountryName string `json:"country_name"` // Country name        text
	Email       string `json:"email"`        // Email               text or empty
	Sub         string `json:"sub"`          // Subject             text or empty
	Com         string `json:"com"`          // Comment             text (includes escaped HTML) or empty
	Tim         int64  `json:"tim"`          // Renamed filename    UNIX timestamp + microseconds
	FileName    string `json:"filename"`     // Original filename   text
	Ext         string `json:"ext"`          // File extension      .jpg, .png, .gif, .pdf, .swf
	Fsize       int    `json:"fsize"`        // File size           1-8388608
	Md5         []byte `json:"md5"`          // File MD5            byte slice
	Width       int    `json:"w"`            // Image width         1-10000
	Height      int    `json:"h"`            // Image height        1-10000
	TnW         int    `json:"tn_w"`         // Thumbnail width     1-250
	TnH         int    `json:"tn_h"`         // Thumbnail height    1-250
	FileDeleted int    `json:"filedeleted"`  // File deleted?       0 (no), 1 (yes)
	Spoiler     int    `json:"spoiler"`      // Spoiler image?      0 (no), 1 (yes)
}

// Only used for unmarshaling json. Must be exported for encoding/json, but don't use this.
type JsonThread struct {
	Posts []*JsonPost `json:posts`
}

// A Post represents all of the attributes of a 4chan post, organized in a more logical fashion.
type Post struct {
	// Post info
	Id       int64
	ThreadId int64
	Sticky   bool
	Closed   bool
	Time     time.Time
	Subject  string

	// Poster info
	Name    string
	Trip    string
	Email   string
	Special string
	Capcode string

	// Country and CountryName are empty unless the board uses country info
	Country     string
	CountryName string

	// Message body
	Comment string

	// File info if any, otherwise nil
	File *File
}

type File struct {
	Id          int64  // Id is what 4chan renames images to (UNIX + microtime, e.g. 1346971121077)
	Name        string // Original filename
	Ext         string
	Size        int
	MD5         []byte
	Width       int
	Height      int
	ThumbWidth  int
	ThumbHeight int
	Deleted     bool
	Spoiler     bool
}

func (self *Post) String() (s string) {
	s += fmt.Sprintf("#%d %s%s on %s:\n", self.Id, self.Name, self.Trip, self.Time.Format(time.RFC822))
	if self.File != nil {
		s += fmt.Sprintf("File: %s%s (%dx%d, %d bytes, md5 %x)\n", self.File.Name, self.File.Ext, self.File.Width, self.File.Height, self.File.Size, self.File.MD5)
	}
	s += self.Comment
	return
}

// A Thread represents a thread of posts.
type Thread []*Post

// Only make one request per second
var cooldown <-chan time.Time

// Request a thread from the API. board is just the board name, without the surrounding slashes.
func GetThread(board string, thread_id int, use_ssl bool) (Thread, error) {
	var url string
	if use_ssl {
		url = fmt.Sprintf("https://api.4chan.org/%s/res/%d.json", board, thread_id)
	} else {
		url = fmt.Sprintf("http://api.4chan.org/%s/res/%d.json", board, thread_id)
	}

	if cooldown != nil {
		<-cooldown
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	cooldown = time.After(1 * time.Second)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ParseReader(resp.Body)
}

// Parse a thread from an io.Reader.
func ParseReader(r io.Reader) (Thread, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return Parse(data)
}

// Parse a thread.
func Parse(data []byte) (Thread, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("No input to process")
	}

	var t JsonThread

	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}

	thread := make(Thread, len(t.Posts))
	for k, v := range t.Posts {
		thread[k] = &Post{
			Id:          v.No,
			ThreadId:    v.Resto,
			Sticky:      v.Sticky == 1,
			Closed:      v.Closed == 1,
			Time:        time.Unix(v.Time, 0),
			Name:        v.Name,
			Trip:        v.Trip,
			Special:     v.Id,
			Capcode:     v.Capcode,
			Country:     v.Country,
			CountryName: v.CountryName,
			Email:       v.Email,
			Subject:     v.Sub,
			Comment:     v.Com,
		}
		if len(v.FileName) > 0 {
			thread[k].File = &File{
				Id:          v.Tim,
				Name:        v.FileName,
				Ext:         v.Ext,
				Size:        v.Fsize,
				MD5:         v.Md5,
				Width:       v.Width,
				Height:      v.Height,
				ThumbWidth:  v.TnW,
				ThumbHeight: v.TnH,
				Deleted:     v.FileDeleted == 1,
				Spoiler:     v.Spoiler == 1,
			}
		}
	}

	return thread, nil
}

// Returns the OP post. Logically this should be the first post, but the spec indicates OP by setting field "resto" to 0.
func (self Thread) OP() *Post {
	for _, p := range self {
		if p.ThreadId == 0 {
			return p
		}
	}
	return nil
}

func (self Thread) Id() int64 {
	return self.OP().Id
}

func (self Thread) String() (s string) {
	for _, post := range self {
		s += post.String() + "\n\n"
	}
	return
}
