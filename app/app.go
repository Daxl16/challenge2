package app

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type distancia struct {
	DistanciaTotal int `json:"distance"`
}

type Prefixes struct {
	//SyncToken    []Prefix   `json: "syncToken"`
	//CreateDate   []Prefix   `json: "createDate"`
	Prefixes     []Prefix   `json:"prefixes"`
	IPv6Prefixes []Prefixv6 `json:"ipv6_prefixes"` //Lo que esta de color es como se lee en el archivo JSON
}

type Prefix struct {
	Ip_prefix            string `json:"ip_prefix"` //Notar que si cambio los parametros json, no los lee
	Region               string `json:"region"`
	Service              int    `json:"service"`
	Network_border_group string `json:"network_border_group"` //Lo que esta de color es como se lee en el archivo JSON
}
type Prefixv6 struct {
	Ipv6_prefix          string `json:"ipv6_prefix"` //Notar que si cambio los parametros json, no los lee
	Region               string `json:"region"`
	Service              int    `json:"service"`
	Network_border_group string `json:"network_border_group"` //Lo que esta de color es como se lee en el archivo JSON
}

type country struct {
	CountryCode0    string `json:"countryCode"`
	CountryCode3    string `json:"countryCode3"`
	CountryCodeName string `json:"countryName"`
}

//Structs usado por FuncList
type IP struct {
	Items []IPSaved
}

type IPSaved struct {
	IPpart string
	Pais   string
	Dist   int
	Invoc  int
}

func Start() {

	// Genera el Slice en el que se van a guardar las IPs
	items := []IPSaved{}
	intprot := IP{items}
	//

	// Comienzo de enrutador Gin
	router := gin.Default()

	// Comienzo de localhost:8080/analyze  ---------> Analiza la IP ingresada de la cual obtiene todos los campos requeridos.
	router.GET("/analyze", func(c *gin.Context) {
		ip := c.Query("ip")

		//FuncGEO
		name, code0 := FuncGEO(ip) // Obtiene nombre y Codigo ISO

		//FuncAWS
		awscon := FuncAWS(ip) // Obtiene si es AWS o no

		//FuncDist
		distancia := FuncDist(name) // Obtiene la distancia

		c.JSON(200, gin.H{
			"IP: ": ip,
			//"Fecha Actual"
			"Pais: ":              name,
			"ISO Code: ":          code0,
			"Distancia estimada ": distancia,
			"Pertenece a AWS: ":   awscon,
		})

		intprot.Items = FuncList(intprot, ip, name, distancia)

	})
	//

	// Comienzo de localhost:8080/stats  ---------> Lee todo el Slice para así poder mostrar todas las IPs analizadas hasta el momento
	// En caso de que una IP haya consultado el servicio varias veces, claramente no va a mostrar la IP por cada intento, sino que muestra la cantidad de Invocaciones por la estructura que le dí al Slice
	router.GET("/stats", func(c *gin.Context) {

		for i := 0; i < len(intprot.Items); i++ {
			c.JSON(200, gin.H{
				"IP: ":           intprot.Items[i].IPpart,
				"Pais: ":         intprot.Items[i].Pais,
				"Distancia: ":    intprot.Items[i].Dist,
				"Invocaciones: ": intprot.Items[i].Invoc + 1,
			})
		}

	})
	//

	// Comienzo de localhost:8080/distlej  ---------> Analiza y lee la IP mas lejana que haya consultado el servicio
	router.GET("/distlej", func(c *gin.Context) {
		compDist := 0
		compInvoc := 0
		Num := 0
		for i := 0; i < len(intprot.Items); i++ { // Compara las IPs distintas en el Slice llamado "intprot.Items", y unicamente guarda el valor de LA DISTANCIA mas lejana
			//fmt.Println(i)
			if compDist < intprot.Items[i].Dist {
				//fmt.Println(intprot.Items[i].Dist)
				compDist = intprot.Items[i].Dist

			}

		}

		for i := 0; i < len(intprot.Items); i++ { // Acá habiendo guardado la distancia previamente. Unicamente busca aquellas IPs en "intprot.Items" que tenga esa distancia, y se guarda la posición "NUM" de la que tenga mas Invoc
			if compDist == intprot.Items[i].Dist {
				if compInvoc < intprot.Items[i].Invoc {
					compInvoc = intprot.Items[i].Invoc
					Num = i
				}
			}
		}

		c.JSON(200, gin.H{ // Muestra la IP mas lejana con mas Invoc de todo el Slice
			"La IP mas lejana fue: ": intprot.Items[Num].IPpart,
			"Radicado en: ":          intprot.Items[Num].Pais,
			"A una distancia de: ":   intprot.Items[Num].Dist,
			"Invocaciones":           intprot.Items[Num].Invoc + 1,
		})
	})

	// Comienzo de localhost:8080/distcerc  ---------> Analiza y lee la IP mas cercana que haya consultado el servicio
	router.GET("/distcer", func(c *gin.Context) {
		comp := 999999
		Num := 0
		for i := 0; i < len(intprot.Items); i++ { // Busca la IP dentro del slice con menos distancia de toda la lista
			fmt.Println(i)
			if comp > intprot.Items[i].Dist {
				fmt.Println(intprot.Items[i].Dist)
				comp = intprot.Items[i].Dist
				Num = i
			}

		}
		fmt.Println(comp)
		c.JSON(200, gin.H{
			"La IP mas cercana fue: ": intprot.Items[Num].IPpart,
			"Radicado en: ":           intprot.Items[Num].Pais,
			"A una distancia de: ":    intprot.Items[Num].Dist,
			"Invocaciones":            intprot.Items[Num].Invoc + 1,
		})
	})
	router.Run(":8080")
	//Termina el enrutador Gin
}

//

// ACA EMPIEZAN LAS FUNCIONES
// De haber tenido mas tiempo, me hubiera gustado ordenar mejor las funciones y ocultarlas en carpetas subsecuentes para así poder dar una mejor presentación
func FuncGEO(ipGET string) (x string, y string) {
	//Comienza a corroborar la url para sacar la información de las IP
	var s = ipGET
	ipnew := strings.Split(s, "/")[0] //Le saca lo que sigue despues del / para poder leerlo

	url := "https://api.ip2country.info/ip?" + ipnew

	spaceClient := http.Client{
		Timeout: time.Second * 2, // Timeout after 2 seconds
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("User-Agent", "spacecount-tutorial")

	res, getErr := spaceClient.Do(req)
	if getErr != nil {
		log.Fatal(getErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	country1 := country{}
	jsonErr := json.Unmarshal(body, &country1)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	return country1.CountryCodeName, country1.CountryCode0

	//Devuelve la info de la IP correspondientes
}

func FuncAWS(ipGET string) (x bool) {
	var xy int
	// Descomprime y abre el JSON con las IP asociadas a AWS
	jsonFile, err := os.Open("ipranges.json")

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Successfully Opened ipranges.json")

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var prefixes Prefixes

	json.Unmarshal(byteValue, &prefixes)
	// Termina de descomprimir el JSON con IPs asociadas a AWS

	//Lee el JSON para ver si la IP esta en la lista de AWS UNICAMENTE
	cont := 0
	for i := 0; i < len(prefixes.Prefixes); i++ {
		if myip := ipGET; myip == prefixes.Prefixes[i].Ip_prefix { //Lee el JSON buscando la IPv4 que pueda estar en el archivo

			//fmt.Println("IP: ", myip)      //La consola avisa que IP se ingresó
			//fmt.Println("Pertenece a AWS") //La consola avisa que esa IP pertenece a una AWS
			//var s = "2600:1f70:4000::/56"
			//ip := strings.Split(s, "/")[0] //Lo que hace es sacarle el /32 a todas las direcciones IP del JSON AWS. Porque sino la pagina web que sigue no las lee
			//fmt.Println(i)
			xy = 1

		}

	}
	if cont == 0 {
		for i := 0; i < len(prefixes.IPv6Prefixes); i++ {
			if myip := ipGET; myip == prefixes.IPv6Prefixes[i].Ipv6_prefix { //Es el mismo mecanismo que en el caso anterior, pero con IPsv6

				//fmt.Println("IP: ", myip)      //La consola avisa que IP se ingresó
				//fmt.Println("Pertenece a AWS") //La consola avisa que esa IP pertenece a una AWS
				//var s = "2600:1f70:4000::/56"
				//ip := strings.Split(s, "/")[0] //Lo que hace es sacarle el /32 a todas las direcciones IP del JSON AWS. Porque sino la pagina web que sigue no las lee
				//fmt.Println(i)
				xy = 1

			}

		}
	}

	if xy == 1 {
		return true //Devuelve verdadero
	} else {
		return false //Devuelve falso
	}
}

func FuncDist(nameGET string) (x int) {
	//router1 := gin.Default()

	if strings.Contains(nameGET, " ") { //En caso de que el nombre del pais de la IP tenga un espacio ("United States" por ejemplo) reemplaza el espacio con un "%20" para que la pagina lo pueda leer correctamente
		nameGET = strings.Replace(nameGET, " ", "%20", -1)
		//fmt.Println(nameGET)
	}

	var yx int

	//router1.GET("/analyze", func(c *gin.Context) {

	countryMain := "Argentina"
	countryTo := nameGET

	url := "https://www.distancia.co/route.json?stops=" + countryMain + "|" + countryTo

	spaceClient := http.Client{
		Timeout: time.Second * 2, // Se cierra luego de 2 segundos sin haber recibido respuesta
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("User-Agent", "spacecount-tutorial")

	res, getErr := spaceClient.Do(req)
	if getErr != nil {
		log.Fatal(getErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	distancia1 := distancia{} //Obtiene la distancia del archivo JSON que se habre y se lee en de la pagina www.distancia.co
	jsonErr := json.Unmarshal(body, &distancia1)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	yx = distancia1.DistanciaTotal //Guarda la distancia y la devuelve

	//c.JSON(200, gin.H{
	//	"La distancia entre ambos paises en KM es de ": distancia1.DistanciaTotal,
	//})

	//})

	//router1.Run(":8080")

	return yx

	//Devuelve la info de la IPs correspondientes
}

//copiar aca todas las funciones para así poder correr los tests unitarios y no complicarme la vida
//
//Testing

func FuncList(intprotGET IP, ipGET string, nameGET string, distanciaGET int) []IPSaved {

	//Agarra los valores y los guarda

	//UNICAMENTE SE PUEDE SUMAR MEDIANTE APPEND. SINO NO ASIGNA UNA CAPACITY (PROPIEDAD DEL SLICE) MAS Y CRASHEA

	itemToAdd := IPSaved{IPpart: ipGET, Pais: nameGET, Dist: distanciaGET}

	length := len(intprotGET.Items) //Explicación abajo
	if length == 0 {
		intprotGET.AddItem(itemToAdd)
		fmt.Println("Lo sumé de una")
	} else { //Hasta acá todo bien
		mod := 0
		for i := 0; i < length; i = i + 1 { //Porque no pongo len(intprot.Items) directamente? Porque si se va modificando sobre la marcha, va a aumentar el contador mientras el For esta funcionando. Por eso se lo asigno a una variable antes para que quede estatico
			if itemToAdd.IPpart == intprotGET.Items[i].IPpart {
				intprotGET.Items[i].Invoc++
				fmt.Println("Sume un invoc")
				mod = 1
			}
		}
		if mod == 0 {
			intprotGET.AddItem(itemToAdd)
			fmt.Println("Sumé una nueva IP!")
		}
	}
	//Termina de agarrar el valor y de guardarlo mediante la funcion AddItem

	return intprotGET.Items
}

func (intprot *IP) AddItem(item IPSaved) []IPSaved {
	intprot.Items = append(intprot.Items, item)
	return intprot.Items
}
