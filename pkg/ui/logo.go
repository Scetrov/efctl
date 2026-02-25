package ui

import "github.com/pterm/pterm"

const LogoASCII = `
                                                  
                       ░ ░░░                      
                   ░░███░▒░███░░                  
                 ░ ░ ░░█▓▒░░░ ░░ ░░               
                ▒░ ▓█░░░░░░░░░   ░▒ ░             
              ░ ░ ▓    ░░█░░    ██░ ░▒            
            ░░░ ░  ░ ▒ █░    ▒ ░  ░ ░             
             ░░ ░  ▒▒░ ▒  █▓ ▓▒   ░ ░░░           
             ░▒ ░░   ▒   ▓░  ▓  ▓ ░               
            ░ ░ ░            █    ░               
                 ▒▒▒           ▒▓▒                
                      ░  ▓░ ░                     
                        ░ ░                       
`

func PrintBanner() {
	pterm.DefaultCenter.Println(pterm.NewRGB(255, 116, 0).Sprint(LogoASCII))
}
