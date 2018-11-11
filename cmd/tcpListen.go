package main

func main() {
	err := make(chan bool)
	close(err)
	close(err)
}
