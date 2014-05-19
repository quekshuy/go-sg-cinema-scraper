
package gv

import (
    "time"
    "log"
    "strings"

    gq "github.com/PuerkitoBio/goquery"

    "github.com/quekshuy/go-sg-cinema-scraper/data"
)

const (
    BASE = "http://www.gv.com.sg"
    URL_CINEMAS = "/cinemas.jsp"
)

// getOrchestrateChanFromContext is a helper function. Since we pass the signalling
// channel via []interface{}, we need to do type assertions to retrieve the original
// channel.
func getOrchestrateChanFromContext(context interface{}) chan int {
    if c, ok := context.([]interface{}); ok {
        channel := c[1]
        if s, ok:=channel.(chan int); ok {
            return s
        }
    }
    return nil
}

// syncCinemaFound sends a +1 into the channel. The channel then behaves like a 
// counting semaphore.
func syncCinemaFound(sync chan int) {
    if sync != nil {
        sync <- 1
    }
}

// syncCinemaDone sends a -1 into the channel. The channel then behaves like a 
// counting semaphore.
func syncCinemaDone(sync chan int) {
    if sync != nil {
        sync <- -1
    }
}


// startGVScrape will begin scraping the GV website for cinemas.
// For each cinema, it runs a goroutine to fetch the movie showtimes. The goroutine
// is run using nextFunc, a ScraperStrategy. Cinemas and Movie objects are returned via
// the channels. The gvCount channel is for synchronization. Whenever we encounter a 
// cinema link that we can proceed to scrape, we send an integer 1 into the channel.
// This is to mark the number of goroutines that are currently running.
func startGVScrape(startUrl string, nextFunc data.ScraperStrategy, cinemas chan<- []*data.Cinema, movies chan<- []*data.Movie, gvCount chan int) {

    var doc *gq.Document
    var err error

    if doc, err = gq.NewDocument(startUrl); err != nil {
        log.Fatal(err)
    }

    // Find all the cinemas
    doc.Find("a.movie").Each(func(i int, s *gq.Selection) {
        if url, ok := s.Attr("href"); ok {
            if text, err := s.Html(); err != nil {
                log.Fatalf("Error getting cinema name for GV: %v", err)
            } else {
                /*log.Println("Going to scrape: " + BASE + "/" + url + ", with name = " + text)*/
                if text != "" {
                    go nextFunc(BASE + "/" + url, []interface{}{text, gvCount}, cinemas, movies)
                    syncCinemaFound(gvCount)
                }
            }
        }
    })
}

// scrapeCinemaMovies parses the layout for a GV cinema's page. Basically it reads a table,
// that GV will display on their page, and parses it for the relevant information. 
// It doesn't have all the information, but that's OK for this really early version.
// context is where calling functions can pass additional info or even function pointers in.
// I haven't worked out how to make it really modular yet, maybe in future we'll include a 
// function that we will call, that's actually a closure, so state can be maintained.
func scrapeCinemaMovies(url string, context interface{}, cinemas chan<- []*data.Cinema, movies chan<- []*data.Movie) {

    var doc *gq.Document
    var err error
    var cinemaName interface{}
    var signalChan chan int

    if ctxList, ok := context.([]interface{}); ok {
        cinemaName = ctxList[0]
        signalChan = getOrchestrateChanFromContext(context)
    }

    log.Println("Retrieving document for " + url)
    if doc, err = gq.NewDocument(url); err != nil {
        log.Fatal(err)
    }

    cinemaMovies := []*data.Movie{}

    if name, ok := cinemaName.(string); ok {

        doc.Find("table tr").Each(func(i int, s *gq.Selection) {

            showtimes := []string{}
            var movieTitle string

            s.Children().Each(func(j int, c *gq.Selection) {

                    var singleTime string

                    if j == 0 {

                        // the 2nd td contains movie name
                        movieTitle, err = c.Find("a.showtimes").Html()

                    } else if j == 2 {
                        // the 4th td contains show times
                        c.Find("a.showtimes").Each(func(i int, a *gq.Selection) {
                            singleTime, err = a.Html()
                            showtimes = append(showtimes, singleTime)
                        })
                    }
            })

            movie := &data.Movie{
                Title: movieTitle,
                CinemaName: name,
                ShowTimes: showtimes,
                // no description and duration here
            }
            /*log.Printf("Found movie: %v\n", *movie)*/

            cinemaMovies = append(cinemaMovies, movie)

        })

        // Create the cinema and send
        cinema := &data.Cinema{
            Name: name,
            Movies: cinemaMovies,
        }

        cinemas <- []*data.Cinema{ cinema }
    }

    // Send the movie details for this cinema
    movies <- cinemaMovies

    // Signal that we are done with one cinema
    signalChan<- -1
}


// prepScrapeCinemaMovies prepares the actual URL for movie showtimes at a particular cinema, then
// calls the actual scraping function.
func prepScrapeCinemaMovies(url string, context interface{}, cinemas chan<- []*data.Cinema, movies chan<- []*data.Movie) {

    var doc *gq.Document
    var err error

    log.Println("Retrieving document for " + url)
    if doc, err = gq.NewDocument(url); err != nil {
        log.Fatal(err)
    }

    allText, err := doc.Html()
    startIdx := strings.Index(allText, "buyTickets2")

    if startIdx >-1 {

        locIdx := strings.Index(allText[startIdx:], "loc=")
        endLoc := strings.Index(allText[startIdx+locIdx:], "&")
        loc := allText[startIdx+locIdx+4 : startIdx+locIdx+endLoc]


        go scrapeCinemaMovies(BASE + "/buyTickets2.jsp?loc="+loc+"&date=" + time.Now().Format("02-01-2006"), context, cinemas, movies)

    } else {
        log.Fatalf("No available source URL")
    }

}

// Load is a non-blocking call that will return via the channels the cinemas and movies
// for this cinema. Notice the signal channel. A true will be sent down the channel when
// Load detects all possible cinemas have been scraped.
func Load(cinemas chan<- []*data.Cinema, movies chan<- []*data.Movie, signal chan interface{}) {

    log.Println("Loading GV")
    countChan := make(chan int)
    go startGVScrape(BASE+URL_CINEMAS, prepScrapeCinemaMovies, cinemas, movies, countChan)

    cinemaCount := 0

    // now we wait for all the cinemas to be returned
    go func() {
        for {
            select {
            case num :=<-countChan:
                cinemaCount += num
                log.Printf("\t\tcinemaCount = %d\n", cinemaCount)
                if cinemaCount == 0 {
                    log.Println("No more cinemas")
                    // signal that it's the end
                    signal<-true
                    log.Println("Signalled that is complete")
                    return
                }
            case <-time.After(time.Second*20):
                //Timeout after 20 seconds of not hearing anything
                log.Println("GV scraping timed out. No new cinemas received")
                return
            }
        }
    }()
}
