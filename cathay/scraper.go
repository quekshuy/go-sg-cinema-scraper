package cathay

import (
    "log"
    "strings"
    "strconv"

    gq "github.com/PuerkitoBio/goquery"

    "github.com/quekshuy/go-sg-cinema-scraper/data"
)

const (
    BASE = "http://www.cathaycineplexes.com.sg"
    URL_MOVIES = "/movie-listing.aspx"
)

type movieUrl struct {
    Title string
    Url string
}

func moviesList(movieDetailsChan chan<- *movieUrl) {

    var doc *gq.Document
    var err error

    log.Println("moviesList")
    doc, err = gq.NewDocument(BASE + URL_MOVIES)
    if err != nil {
        log.Fatalf("Error getting Cathay movie listings: %v\n", err)
    }

    doc.Find("#promoboxcontainer table a.title").Each(func(i int, s *gq.Selection) {
        if movieName, err := s.Html(); err != nil {
            log.Fatalf("Cathay: Error getting movie title: %v\n", err)
        } else {
            if href, ok := s.Attr("href"); ok {
                m := &movieUrl{
                    Title: movieName,
                    Url: href,
                }
                movieDetailsChan<-m
            }
        }
    })
    log.Println("end MoviesList")
    close(movieDetailsChan)
}

func getMovieTimes(s *gq.Selection) []string {

    showTimes := []string{}
    s.Find("#showtimeitem_time a.cine_time").Each(func(i int, q *gq.Selection) {
        showtime, _ := q.Html()
        if showtime != "" {
            showTimes = append(showTimes, showtime)
        }
    })
    return showTimes
}

func getMovieDuration(doc *gq.Document) int {

    var dur int

    doc.Find("span#ctl00_cphContent_lblRuntime").Each(func(i int, s *gq.Selection) {
        time, _ := s.Html()
        if time != "" {
            slices := strings.Split(time, " ")
            if len(slices) > 0 {
                dur, _ = strconv.Atoi(slices[0])
            }
        }
    })
    return dur
}

// Movie details page.
func movieDetails(
    mu *movieUrl,
    signalChan chan<- int,
    cinemaChan chan<- *data.Cinema,
    movieChan chan<- *data.Movie,
){

    // We got the title
    // Now find the run time
    var doc *gq.Document
    var err error

    doc, err = gq.NewDocument(BASE + "/" + mu.Url)
    if err != nil {
        log.Fatalf("Cathay: Error getting movie details: %s", BASE + "/" + mu.Url)
    }

    m := &data.Movie{ Title: mu.Title }
    m.Duration = getMovieDuration(doc)

    // we could use a better selector (i.e. :first) but i think this might be safer
    doc.Find("div#mdetails_synopsis p").Each(func(i int, s *gq.Selection) {
        if i == 0 {
            desc, _ := s.Html()
            m.Description = desc
        }
    })


    c := &data.Cinema{}
    doc.Find("#showtimes #cinema_name").Each(func(i int, s *gq.Selection){

        m.ShowTimes = getMovieTimes(s.Next())

        name, err := s.Html()
        if err != nil {
            log.Fatalf("Cathay: Error getting the cinema's name\n")
        }
        c.Name = name
    })

    // in the interim, let's temporarily return the cinema with a single
    // movie. 
    c.Movies = []*data.Movie{ m }

    // The receiver of cinemaChan needs to aggregate the cinemas
    cinemaChan <- c
    movieChan <- m
    signalChan<-1
}

func waitForCinema(cinemaChan <-chan *data.Cinema, mainCinemaChan chan<- []*data.Cinema) {

    cMap := make(map[string]*data.Cinema)

    for c:= range cinemaChan {
        if cinema := cMap[c.Name]; cinema != nil {
            // add to the cinema.Movies
            if c.Movies != nil && c.Movies[0] != nil {
                cinema.Movies = append(cinema.Movies, c.Movies[0])
            }
        } else {
            cMap[c.Name] = c
        }
    }

    // make into a list
    cinemas := []*data.Cinema{}

    for _, v := range cMap {
        cinemas = append(cinemas, v)
    }

    // send the list out
    mainCinemaChan <- cinemas
    // close the channel
    close(mainCinemaChan)
}

func waitForMovies( moviesChan <-chan *data.Movie, mainMoviesChan chan<- []*data.Movie) {

    collect := []*data.Movie{}

    for m := range moviesChan {
        collect = append(collect, m)
    }

    // send to main channel
    mainMoviesChan <- collect
    close(mainMoviesChan)
}

func Load(cinemas chan<- []*data.Cinema, movies chan<- []*data.Movie) {

    movieUrlsChan := make(chan *movieUrl)
    movieChan := make(chan *data.Movie)
    cinemaChan := make(chan *data.Cinema)

    go moviesList(movieUrlsChan)

    // moviesList will send movieUrl objects which
    // this goroutine will collect and run subsequent stuff for.
    go func() {

        signalChan := make(chan int)
        muList := []*movieUrl{}

        // collect all movieUrls first 
        // so that we have a stableNumber for totalMu
        for mu := range movieUrlsChan {
            muList = append(muList, mu)
        }

        totalMu := len(muList)
        for _, mu := range muList {
            go movieDetails(mu, signalChan, cinemaChan, movieChan)
        }

        // this goroutine will enable processing to continue
        // after all movie details have been parsed.
        go func() {

            for _ = range signalChan {
                totalMu -= 1
                if totalMu == 0 {
                    // close the channels
                    // and break
                    log.Println("Closing channels")
                    close(movieChan)
                    close(cinemaChan)
                }
            }
        }()


        go waitForMovies(movieChan, movies)
        go waitForCinema(cinemaChan, cinemas)
    }()
}
