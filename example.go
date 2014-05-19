package main

import (
    "log"
    "time"
    "github.com/quekshuy/go-sg-cinema-scraper/data"
    /*"github.com/quekshuy/go-sg-cinema-scraper/gv"*/
    "github.com/quekshuy/go-sg-cinema-scraper/cathay"
)

func waitForCinema(cinemaChan chan []*data.Cinema) {
    for {
        select {
        case <-cinemaChan:
            case cinemas :=<-cinemaChan:
                for _, c:= range cinemas {
                    log.Printf("Cinema: %v\n", *c)
                }
            case <-time.After(time.Second * 20):
                log.Println("Timeout cinemas")
                return
        }
    }
}

func waitForMovie(moviesChan chan[]*data.Movie, signal chan interface{}) {
    for {
        select {
        case <-moviesChan:
        case movies :=<-moviesChan:
            for _, m:=range movies{
                log.Printf("Movie: %v\n", *m)
            }
        case <-time.After(time.Second *20):
            log.Println("Timeout movies")
            return
        }
    }
}

func main() {
    log.Println("Starting")
    cinemaChan := make(chan []*data.Cinema)
    moviesChan := make(chan []*data.Movie)
    signal := make(chan interface{})

    /* FOR GV
    go waitForCinema(cinemaChan)
    go waitForMovie(moviesChan, signal)

    gv.Load(cinemaChan, moviesChan, signal)
    // this signals that scraping is done and we can
    // all go home.
    <-signal
    */

    // For Cathay
    go func() {
        for movies := range moviesChan {
            for _, m := range movies {
                log.Printf("Movie: %v\n", m)
            }
        }
        signal<-true
    }()

    go func() {

        for cinemas := range cinemaChan {
            for _, c := range cinemas {
                log.Printf("Cinema: %v\n", c)
            }
        }
        signal<-true

    }()

    cathay.Load(cinemaChan, moviesChan)

    // 2 signals marks as finished
    <-signal
    <-signal
}
