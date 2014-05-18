
package gv

import (
    //"fmt"
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


func startGVScrape(startUrl string, nextFunc data.ScraperStrategy, cinemas chan<- []*data.Cinema, movies chan<- []*data.Movie, signal chan interface{}) {

    log.Println("Started gv with " + startUrl)
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
                    go nextFunc(BASE + "/" + url, []interface{}{text, signal}, cinemas, movies)
                }
            }
        }
    })
}

func scrapeCinemaMovies(url string, context interface{}, cinemas chan<- []*data.Cinema, movies chan<- []*data.Movie) {

    var doc *gq.Document
    var err error
    var cinemaName interface{}
    /*var signal chan interface{}*/

    if ctxList, ok := context.([]interface{}); ok {
        cinemaName = ctxList[0]
        /*if s, ok := ctxList[1].(chan interface{}); ok {*/
            /*[>signal = s<]*/
        /*}*/
        /*switch s := ctxList[1].(type) {*/
        /*case chan interface{}:*/
            /*signal = s*/
        /*case nil:*/
            /*log.Println("Nope not a chan")*/
        /*default:*/
            /*log.Println("default case")*/
        /*}*/

        /*if s, ok := ctxList[1].(chan interface{}); ok {*/
            /*log.Println("assigned signal")*/
            /*signal = s*/
        /*}*/
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
            log.Printf("Found movie: %v\n", *movie)

            cinemaMovies = append(cinemaMovies, movie)

        })

        // Create the cinema and send
        cinema := &data.Cinema{
            Name: name,
            Movies: cinemaMovies,
        }
        log.Printf("Found cinema: %v\n", *cinema)
        cinemas <- []*data.Cinema{ cinema }
    }

    // Send the movie details
    movies <- cinemaMovies
    // Signal that we are done
    /*if signal != nil {*/
        /*signal <- true*/
    /*}*/

}


// Only scrapes for today
func prepScrapeCinemaMovies(url string, context interface{}, cinemas chan<- []*data.Cinema, movies chan<- []*data.Movie) {

    var doc *gq.Document
    var err error

    log.Println("Retrieving document for " + url)
    if doc, err = gq.NewDocument(url); err != nil {
        log.Fatal(err)
    }

    allText, err := doc.Html()
    // find the buyTickets2.jsp
    // and print it out
    startIdx := strings.Index(allText, "buyTickets2")
    if startIdx >-1 {
        /*log.Println("Found buyTickets2: " + allText[startIdx:])*/
        locIdx := strings.Index(allText[startIdx:], "loc=")
        /*log.Println("Found loc : "+allText[startIdx+locIdx:])*/
        endLoc := strings.Index(allText[startIdx+locIdx:], "&")
        loc := allText[startIdx+locIdx+4 : startIdx+locIdx+endLoc]
        log.Println("Location: " + loc)
        /*locIdx := strings.Index(allText[startIdx:], "loc=")*/
        /*locEnd := strings.Index(allText[startIdx+locIdx:], "&")*/
        /*loc := allText[locIdx+4:locEnd]*/
        /*log.Println("Buytickets loc = " + loc)*/
        go scrapeCinemaMovies(BASE + "/buyTickets2.jsp?loc="+loc+"&date=" + time.Now().Format("02-01-2006"), context, cinemas, movies)

    } else {
        log.Fatalf("No available source URL")
    }

    // we have to hunt for the ajax call
    // which is actually BASE + /buyTickets2.jsp?loc=<some_int>&date=<dd-mm-yyyy>
    /*firstElem := doc.Find("div#tabPanelWrapper div.tabPanel ul li a").First()*/
    /*// empty *Selection if does not exist*/
    /*if firstElem.Nodes != nil {*/
        /*// Find the URL we want*/
        /*if onClick, exists := firstElem.Attr("onclick"); exists {*/
            /*log.Println("onclick = " + onClick)*/
            /*startIdx := strings.Index(onClick, "buyTickets2")*/
            /*if startIdx > -1 {*/
                /*endIdx := strings.Index(onClick[startIdx:], "'")*/
                /*nextUrl := onClick[startIdx:startIdx+endIdx]*/
                /*go scrapeCinemaMovies(BASE+"/"+nextUrl, context, cinemas, movies)*/
            /*}*/
        /*}*/
    /*}*/
}

func Load(cinemas chan<- []*data.Cinema, movies chan<- []*data.Movie, signal chan interface{}) {

    log.Println("Loading GV")
    go startGVScrape(BASE+URL_CINEMAS, prepScrapeCinemaMovies, cinemas, movies, signal)
}
