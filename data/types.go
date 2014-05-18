
package data

type Movie struct {
    Duration int // in mins
    Description string
    Title string
    CinemaName string
    ShowTimes []string
}

type Cinema struct {
    Name string
    Movies []*Movie
}

// URL, context for this current URL (basically a nest hierarchy),
// channel for cinemas, channel for movies
type ScraperStrategy func(string, interface{}, chan<- []*Cinema, chan<- []*Movie)
