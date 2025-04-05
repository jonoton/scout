# Profiling with GoLang

## Dependencies
1. Graphviz
    ```
    sudo apt install graphviz
    ```

## How to Run
1. Add to main.go (Skip for scout as it's already done for you)
    ```
    import (
        ...
        
        baseHttp "net/http"
        _ "net/http/pprof"
        
        ...
    )

    func main() {
        ...

        // DEBUG ONLY
        go func() {
            log.Println(baseHttp.ListenAndServe("localhost:6060", nil))
        }()

        ...
    }
    ```
1. Run pprof
    1. CPU
        ```
        go tool pprof -http localhost:8081 http://localhost:6060
        ```
    1. Memory
        ```
        go tool pprof -http localhost:8081 http://localhost:6060/debug/pprof/heap
        ```
    1. gocv Mat
        ```
        Run PROJECT with tag set:
        go run -tags matprofile PROJECT
        ```
        ```
        go tool pprof -http localhost:8081 http://localhost:6060/debug/pprof/gocv.io/x/gocv.Mat
        ```
    1. SharedMat
        ```
        Run PROJECT with tag set:
        go run -tags profile PROJECT
        ```
        ```
        go tool pprof -http localhost:8081 http://localhost:6060/debug/pprof/github.com/jonoton/go-sharedmat.counts
        ```
        See `scout/main_profile.go` and `scout/sharedmat/sharedmat_profile.go` for more insight
        
1. The default web browser will open
1. Navigate the Web UI
    
    Tip: The `View -> Flame Graph` is very handy
