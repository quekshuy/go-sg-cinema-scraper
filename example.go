package main

import (
    "log"
    "time"
    "github.com/quekshuy/go-sg-cinema-scraper/data"
    "github.com/quekshuy/go-sg-cinema-scraper/gv"
)

func waitForCinema(cinemaChan chan []*data.Cinema) {
    for {
        select {
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
        case movies :=<-moviesChan:
            for _, m:=range movies{
                log.Printf("Movie: %v\n", *m)
            }
        case <-time.After(time.Second *20):
            log.Println("Timeout movies")
            signal <- true
            return
        }
    }
}

func main() {
    log.Println("Starting")
    cinemaChan := make(chan []*data.Cinema)
    moviesChan := make(chan []*data.Movie)
    signal := make(chan interface{})

    go waitForCinema(cinemaChan)
    go waitForMovie(moviesChan, signal)

    gv.Load(cinemaChan, moviesChan, signal)
    <-signal
}
