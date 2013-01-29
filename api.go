// Package api pulls 4chan board and thread data from the JSON API into native Go data structures.
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var (
	// Whether or not to use HTTPS for requests.
	SSL bool = false
	// Cooldown time for updating threads using (*Thread).Update().
	// If it is set to less than 10 seconds, it will be re-set to 10 seconds
	// before being used.
	UpdateCooldown time.Duration = 15 * time.Second
	cooldown       <-chan time.Time
)

func get(path string, modify func(*http.Request) error) (*http.Response, error) {
	if SSL {
		path = "https://api.4chan.org" + path
	} else {
		path = "http://api.4chan.org" + path
	}
	if cooldown != nil {
		<-cooldown
	}
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	if modify != nil {
		err = modify(req)
		if err != nil {
			return nil, err
		}
	}

	resp, err := http.DefaultClient.Do(req)
	cooldown = time.After(1 * time.Second)
	return resp, err
}

func get_decode(path string, dest interface{}, modify func(*http.Request) error) error {
	resp, err := get(path, modify)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(dest)
}

// Direct mapping from the API's JSON to a Go type.
type jsonPost struct {
	No            int64  `json:"no"`             // Post number         1-9999999999999
	Resto         int64  `json:"resto"`          // Reply to            0 (is thread), 1-999999999999
	Sticky        int    `json:"sticky"`         // Stickied thread?    0 (no), 1 (yes)
	Closed        int    `json:"closed"`         // Closed thread?      0 (no), 1 (yes)
	Now           string `json:"now"`            // Date and time       MM\/DD\/YY(Day)HH:MM (:SS on some boards)
	Time          int64  `json:"time"`           // UNIX timestamp      UNIX timestamp
	Name          string `json:"name"`           // Name                text or empty
	Trip          string `json:"trip"`           // Tripcode            text (format: !tripcode!!securetripcode)
	Id            string `json:"id"`             // ID                  text (8 characters), Mod, Admin
	Capcode       string `json:"capcode"`        // Capcode             none, mod, admin, admin_highlight, developer
	Country       string `json:"country"`        // Country code        ISO 3166-1 alpha-2, XX (unknown)
	CountryName   string `json:"country_name"`   // Country name        text
	Email         string `json:"email"`          // Email               text or empty
	Sub           string `json:"sub"`            // Subject             text or empty
	Com           string `json:"com"`            // Comment             text (includes escaped HTML) or empty
	Tim           int64  `json:"tim"`            // Renamed filename    UNIX timestamp + microseconds
	FileName      string `json:"filename"`       // Original filename   text
	Ext           string `json:"ext"`            // File extension      .jpg, .png, .gif, .pdf, .swf
	Fsize         int    `json:"fsize"`          // File size           1-8388608
	Md5           []byte `json:"md5"`            // File MD5            byte slice
	Width         int    `json:"w"`              // Image width         1-10000
	Height        int    `json:"h"`              // Image height        1-10000
	TnW           int    `json:"tn_w"`           // Thumbnail width     1-250
	TnH           int    `json:"tn_h"`           // Thumbnail height    1-250
	FileDeleted   int    `json:"filedeleted"`    // File deleted?       0 (no), 1 (yes)
	Spoiler       int    `json:"spoiler"`        // Spoiler image?      0 (no), 1 (yes)
	CustomSpoiler int    `json:"custom_spoiler"` // Custom spoilers?	1-99
	OmittedPosts  int    `json:"omitted_posts"`  // # replies omitted	1-10000
	OmittedImages int    `json:"omitted_images"` // # images omitted	1-10000
	Replies       int    `json:"replies"`        // total # of replies	0-99999
	Images        int    `json:"images"`         // total # of images	0-99999
	BumpLimit     int    `json:"bumplimit"`      // bump limit?			0 (no), 1 (yes)
	ImageLimit    int    `json:"imagelimit"`     // image limit?		0 (no), 1 (yes)
}

// A Post represents all of the attributes of a 4chan post, organized in a more directly usable fashion.
type Post struct {
	// Post info
	Id      int64
	Thread  *Thread
	Time    time.Time
	Subject string

	// These are only present in an OP post. They are exposed through their
	// corresponding Thread getter methods.
	replies        int
	images         int
	omitted_posts  int
	omitted_images int
	bump_limit     bool
	image_limit    bool
	sticky         bool
	closed         bool
	custom_spoiler int // the number of custom spoilers on a given board

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

func (self *Post) String() (s string) {
	s += fmt.Sprintf("#%d %s%s on %s:\n", self.Id, self.Name, self.Trip, self.Time.Format(time.RFC822))
	if self.File != nil {
		s += self.File.String()
	}
	s += self.Comment
	return
}

// File represents an uploaded image.
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

func (self *File) String() string {
	return fmt.Sprintf("File: %s%s (%dx%d, %d bytes, md5 %x)\n", self.Name, self.Ext, self.Width, self.Height, self.Size, self.MD5)
}

// Return the URL of the post's country flag icon
func (self *Post) CountryFlagURL() string {
	if self.Country == "" {
		return ""
	}
	prefix := "http"
	if SSL {
		prefix += "s"
	}
	// lol /pol/
	if self.Thread.Board == "pol" {
		return prefix + "://static.4chan.org/image/country/troll/" + self.Country + ".gif"
	}
	return prefix + "://static.4chan.org/image/country/" + self.Country + ".gif"
}

// A Thread represents a thread of posts.
type Thread struct {
	Posts []*Post
	OP    *Post
	Board string // without slashes ex. "g" or "ic"

	date_recieved time.Time
	cooldown      <-chan time.Time
}

// Get an index of threads from the given board and page.
func GetIndex(board string, page int) ([]*Thread, error) {
	resp, err := get(fmt.Sprintf("/%s/%d.json", board, page), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	threads, err := ParseIndex(resp.Body, board)
	now := time.Now()
	for _, t := range threads {
		t.date_recieved = now
	}
	return threads, err
}

// Request a thread from the API. board is just the board name, without the
// surrounding slashes. If a thread is being updated, use an existing thread's
// Update() method if possible because that uses If-Modified-Since
func GetThread(board string, thread_id int64) (*Thread, error) {
	return get_thread(board, thread_id, time.Unix(0, 0))
}

func get_thread(board string, thread_id int64, stale_time time.Time) (*Thread, error) {
	resp, err := get(fmt.Sprintf("/%s/res/%d.json", board, thread_id), func(req *http.Request) error {
		if stale_time.Unix() != 0 {
			req.Header.Add("If-Modified-Since", stale_time.UTC().Format(time.RFC1123))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	thread, err := ParseThread(resp.Body, board)
	thread.date_recieved = time.Now()

	return thread, err
}

// Convert a JSON response for multiple threads into a native Go data structure
func ParseIndex(r io.Reader, board string) ([]*Thread, error) {
	var t struct {
		Threads []struct {
			Posts []*jsonPost `json:"posts"`
		} `json:"threads"`
	}

	if err := json.NewDecoder(r).Decode(&t); err != nil {
		return nil, err
	}

	threads := make([]*Thread, len(t.Threads))
	for i, json_thread := range t.Threads {
		thread := &Thread{Posts: make([]*Post, len(t.Threads[i].Posts)), Board: board}
		for k, v := range json_thread.Posts {
			thread.Posts[k] = json_to_native(v, thread)
			if v.No == 0 {
				thread.OP = thread.Posts[k]
			}
		}
		// TODO: fix this up
		if thread.OP == nil {
			thread.OP = thread.Posts[0]
		}
		threads[i] = thread
	}

	return threads, nil
}

// Convert a JSON response for one thread into a native Go data structure
func ParseThread(r io.Reader, board string) (*Thread, error) {
	var t struct {
		Posts []*jsonPost `json:"posts"`
	}

	if err := json.NewDecoder(r).Decode(&t); err != nil {
		return nil, err
	}

	thread := &Thread{Posts: make([]*Post, len(t.Posts)), Board: board}
	for k, v := range t.Posts {
		thread.Posts[k] = json_to_native(v, thread)
		if v.No == 0 {
			thread.OP = thread.Posts[k]
		}
	}
	// TODO: fix this up
	if thread.OP == nil {
		thread.OP = thread.Posts[0]
	}

	return thread, nil
}

func json_to_native(v *jsonPost, thread *Thread) *Post {
	p := &Post{
		Id:             v.No,
		sticky:         v.Sticky == 1,
		closed:         v.Closed == 1,
		Time:           time.Unix(v.Time, 0),
		Name:           v.Name,
		Trip:           v.Trip,
		Special:        v.Id,
		Capcode:        v.Capcode,
		Country:        v.Country,
		CountryName:    v.CountryName,
		Email:          v.Email,
		Subject:        v.Sub,
		Comment:        v.Com,
		custom_spoiler: v.CustomSpoiler,
		replies:        v.Replies,
		images:         v.Images,
		omitted_posts:  v.OmittedPosts,
		omitted_images: v.OmittedImages,
		bump_limit:     v.BumpLimit == 1,
		image_limit:    v.ImageLimit == 1,
		Thread:         thread,
	}
	if len(v.FileName) > 0 {
		p.File = &File{
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
	return p
}

// Update an existing thread in-place.
func (self *Thread) Update() (new_posts, deleted_posts int, err error) {
	if self.cooldown != nil {
		<-self.cooldown
	}
	var thread *Thread
	thread, err = get_thread(self.Board, self.Id(), self.date_recieved)
	if UpdateCooldown < 10*time.Second {
		UpdateCooldown = 10 * time.Second
	}
	self.cooldown = time.After(UpdateCooldown)
	if err != nil {
		return 0, 0, err
	}
	var a, b int
	// traverse both threads in parallel to check for deleted/appended posts
	for a, b = 0, 0; a < len(self.Posts); a, b = a+1, b+1 {
		if self.Posts[a].Id == thread.Posts[b].Id {
			continue
		}
		// a post has been deleted, go back one to compare with the next
		b--
		deleted_posts++
	}
	new_posts = len(thread.Posts) - b
	self.Posts = thread.Posts
	return
}

func (self *Thread) Id() int64 {
	return self.OP.Id
}

func (self *Thread) String() (s string) {
	for _, post := range self.Posts {
		s += post.String() + "\n\n"
	}
	return
}

func (self *Thread) Replies() int {
	return self.OP.replies
}
func (self *Thread) Images() int {
	return self.OP.images
}
func (self *Thread) OmittedPosts() int {
	return self.OP.omitted_posts
}
func (self *Thread) OmittedImages() int {
	return self.OP.omitted_images
}
func (self *Thread) BumpLimit() bool {
	return self.OP.bump_limit
}
func (self *Thread) ImageLimit() bool {
	return self.OP.image_limit
}
func (self *Thread) Closed() bool {
	return self.OP.closed
}
func (self *Thread) Sticky() bool {
	return self.OP.sticky
}
func (self *Thread) CustomSpoiler() int {
	return self.OP.custom_spoiler
}

func (self *Thread) CustomSpoilerURL(id int, ssl bool) string {
	if id > self.OP.custom_spoiler {
		return ""
	}
	prefix := "http"
	if ssl {
		prefix += "s"
	}
	return fmt.Sprintf("%s://static.4chan.org/image/spoiler-%s%d.png", prefix, self.Board, id)
}

// Board represents a board as represented on /boards.json
type Board struct {
	Board string `json:"board"`
	Title string `json:"title"`
}

// Board names/descriptions will be cached here after a call to LookupBoard or GetBoards
var Boards []Board

func LookupBoard(name string) Board {
	if Boards == nil {
		_, err := GetBoards()
		if err != nil {
			return Board{}
		}
	}
	for _, b := range Boards {
		if name == b.Board {
			return b
		}
	}
	return Board{}
}

// Get the list of boards.
func GetBoards() ([]Board, error) {
	var b struct {
		Boards []Board `json:"boards"`
	}
	err := get_decode("/boards.json", &b, nil)
	if err != nil {
		return nil, err
	}
	Boards = b.Boards
	return b.Boards, nil
}

type Catalog []struct {
	Page    int
	Threads []*Thread
}

type catalog []struct {
	Page    int         `json:"page"`
	Threads []*jsonPost `json:"threads"`
}

// Get a board's catalog.
func GetCatalog(board string) (Catalog, error) {
	if len(board) == 0 {
		return nil, fmt.Errorf("api: GetCatalog: No board name given")
	}
	var c catalog
	err := get_decode(fmt.Sprintf("/%s/catalog.json", board), c, nil)
	if err != nil {
		return nil, err
	}

	cat := make(Catalog, len(c))
	for i, page := range c {
		extracted := struct {
			Page    int
			Threads []*Thread
		}{page.Page, make([]*Thread, len(page.Threads))}
		for j, post := range page.Threads {
			thread := &Thread{Posts: make([]*Post, 1), Board: board}
			post := json_to_native(post, thread)
			thread.Posts[0] = post
			extracted.Threads[j] = thread
		}
		cat[i] = extracted
	}
	return cat, nil
}
