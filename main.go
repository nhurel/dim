package main

/*import (
	"github.com/nhurel/dim/lib"
	"flag"
	"github.com/Sirupsen/logrus"
)*/
import (
	"fmt"
	"github.com/nhurel/dim/cmd"
	"os"
)

func main() {

	if err := cmd.RootCommand.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

}
