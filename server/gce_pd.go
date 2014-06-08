package server

import (
	"fmt"
	"os/exec"
	"log"
)

func SaveToPd(imgID, pd_path string) error {
	log.Printf("Saving imgID:%v to pd_path:%v", imgID, pd_path)
	return exec.Command("sh", "-c", fmt.Sprintf("docker save %v> /%v/%v", imgID, pd_path, imgID)).Run();
}

func LoadFromPd(imgID, pd_path string) error {
	log.Printf("Loading imgID:%v from pd_path:%v", imgID, pd_path)
	return exec.Command("sh", "-c", fmt.Sprintf("cat /%v/%v | docker load", pd_path, imgID)).Run();
}
