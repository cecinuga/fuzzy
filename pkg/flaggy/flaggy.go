package flaggy

import (
	"fmt"
	"fuzzy/internal/utils"
	"os"
)

type Value interface {
	String() 	string
}

type StringVal string

func (s *StringVal) String() string {
	return string(*s)
}

type Flag struct { 
	usage 		string
	val 		Value
	def 		Value
	validator 	utils.Matcher
}

type Flags map[string]*Flag

// String definisce un flag string con nome, valore di default e usage.
// Ritorna un puntatore alla stringa che conterrà il valore del flag.
func (f *Flags) String(name string, value string, usage string, validator utils.Matcher) *string {
	p := new(string)
	f.StringVar(p, name, value, usage, validator)
	return p
}

// StringVar definisce un flag string con nome, valore di default e usage.
// L'argomento p punta a una variabile string che memorizza il valore del flag.
func (f *Flags) StringVar(p *string, name string, value string, usage string, validator utils.Matcher) {
	if f == nil {
		panic("Flags map is nil")
	}
	
	// Inizializza la variabile con il valore di default
	*p = value
	
	// Crea il StringVal che wrappa il puntatore
	val := (*StringVal)(p)
	def := StringVal(value)
	
	// Crea il flag
	flag := &Flag{
		usage:     usage,
		val:       val,
		def:       &def,
		validator: validator, // Validator di default che accetta tutto
	}
	
	// Registra il flag nella mappa
	(*f)[name] = flag
}

// Parse analizza gli argomenti della command line e popola i flag registrati
func (f *Flags) Parse() {
	f.ParseArgs(os.Args[1:])
}

// ParseArgs analizza gli argomenti forniti e popola i flag registrati
func (f *Flags) ParseArgs(args []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		
		if !isFlag(arg) {
			continue
		}
		
		name := arg[1:] // Rimuove il prefisso '-'
		if len(name) == 0 {
			continue
		}
		
		// Gestisce '--' prefix
		if name[0] == '-' {
			name = name[1:]
		}
		
		flag, exists := (*f)[name]
		if !exists {
			continue // Flag non riconosciuto, ignora
		}
		
		// Per flag string, il prossimo argomento è il valore
		if i+1 < len(args) && !isFlag(args[i+1]) {
			value := args[i+1]
			
			// Valida il valore se c'è un validator
			if flag.validator != nil && !flag.validator(value) {
				f.Help()
				panic("Error Parsing flags")
			}
			
			// Imposta il valore nel flag
			if stringVal, ok := flag.val.(*StringVal); ok {
				*stringVal = StringVal(value)
			}
			
			i++ // Salta il valore che abbiamo appena processato
		}
	}
}

// isFlag controlla se una stringa è un flag (inizia con '-' o '--')
func isFlag(s string) bool {
	return len(s) > 1 && s[0] == '-'
}

// Help stampa il manuale di utilizzo con tutti i flag registrati
func (f *Flags) Help() {
	if f == nil || len(*f) == 0 {
		fmt.Println("No flags defined")
		return
	}

	fmt.Println("Usage:")
	
	// Trova la lunghezza massima dei nomi per allineare l'output
	maxLen := 0
	for name := range *f {
		if len(name) > maxLen {
			maxLen = len(name)
		}
	}
	
	// Stampa ogni flag con formatting uniforme
	for name, flag := range *f {
		if flag == nil {
			continue
		}
		
		// Determina il valore di default
		defaultValue := ""
		if flag.def != nil {
			defaultValue = flag.def.String()
		}
		
		// Format: -name      usage (default: "value")
		padding := maxLen - len(name)
		spaces := ""
		for i := 0; i < padding+2; i++ {
			spaces += " "
		}
		
		fmt.Printf("  -%s%s%s", name, spaces, flag.usage)
		
		if defaultValue != "" {
			fmt.Printf(" (default: \"%s\")", defaultValue)
		}
		
		fmt.Println()
	}
}

// Esempio di utilizzo (simile al package flag standard):
/*
func ExampleUsage() {
	flags := make(Flags)
	
	// Definisci i flag
	endpoint := flags.String("e", "", "API endpoint URL")
	method := flags.String("m", "GET", "HTTP request method")
	dict := flags.String("dict", "", "Dictionary file path")
	
	var insecure bool
	flags.BoolVar(&insecure, "k", false, "Skip TLS certificate verification")
	
	// Mostra l'help se richiesto
	if len(os.Args) == 1 || (len(os.Args) == 2 && os.Args[1] == "-h") {
		flags.Help()
		return
	}
	
	// Parse degli argomenti
	flags.Parse()
	
	// Uso dei valori
	fmt.Printf("Endpoint: %s\n", *endpoint)
	fmt.Printf("Method: %s\n", *method)
	fmt.Printf("Dictionary: %s\n", *dict)
	fmt.Printf("Insecure: %v\n", insecure)
}

// Output esempio del metodo Help():
// Usage:
//   -e       API endpoint URL (default: "")
//   -m       HTTP request method (default: "GET")
//   -dict    Dictionary file path (default: "")
//   -k       Skip TLS certificate verification (default: "false")
*/
