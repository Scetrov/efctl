package ui

import "github.com/pterm/pterm"

const LogoASCII = `
         /\
        //\\
       //  \\
      //____\\
      |      |
      |      |
     /|      |\
    //|______|\ \
      /  \/  \
     /        \

     /==========\
    /            \
   /    efctl     \
   \              /
    \            /
     \==========/
`

func PrintBanner() {
	pterm.DefaultCenter.Println(pterm.NewRGB(255, 116, 0).Sprint(LogoASCII))
}
