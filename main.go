package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

type EnvelopeRequest struct {
	XMLName   xml.Name `xml:"soapenv:Envelope"`
	XmlnsSoap string   `xml:"xmlns:soapenv,attr"`
	XmlnsCli  string   `xml:"xmlns:cli,attr"`
	Header    EnvelopeHeaderRequest
	Body      EnvelopeBodyRequest
}

type EnvelopeHeaderRequest struct {
	XMLName xml.Name `xml:"soapenv:Header"`
}

type EnvelopeBodyRequest struct {
	XMLName     xml.Name `xml:"soapenv:Body"`
	ConsultaSRO consultaSRO
}

type consultaSRO struct {
	XMLName       xml.Name `xml:"cli:consultaSRO"`
	TipoConsulta  string   `xml:"tipoConsulta"`
	TipoResultado string   `xml:"tipoResultado"`
	UsuarioSRO    string   `xml:"usuarioSro"`
	SenhaSRO      string   `xml:"senhaSro"`
	ListaObjetos  []string `xml:"listaObjetos"`
}

type envelopeResult struct {
	XMLName xml.Name           `xml:"Envelope"`
	XMLBody envelopeBodyResult `xml:"Body"`
}

type envelopeBodyResult struct {
	XMLName  xml.Name            `xml:"Body"`
	Consulta consultaSroResponse `xml:"consultaSROResponse"`
}

type consultaSroResponse struct {
	XMLName   xml.Name `xml:"consultaSROResponse"`
	XMLReturn string   `xml:"return"`
}

type rastro struct {
	XMLName xml.Name `xml:"rastro"`
	Versao  string   `xml:"versao"`
	Qtde    uint16   `xml:"qtd"`
	Objetos []objeto `xml:"objeto"`
}

type objeto struct {
	XMLName xml.Name `xml:"objeto"`
	Numero  string   `xml:"numero"`
	Eventos evento   `xml:"evento"`
}

type evento struct {
	XMLName   xml.Name `xml:"evento"`
	Cidade    string   `xml:"cidade"`
	Tipo      string   `xml:"tipo"`
	Status    uint8    `xml:"status"`
	Data      string   `xml:"data"`
	Hora      string   `xml:"hora"`
	Descricao string   `xml:"descricao"`
}

const BucketSize = 5

func main() {
	objetos := readFromCsv()
	loops := len(objetos) / BucketSize
	remainder := len(objetos) % BucketSize
	lastIndex := 0

	for i := 0; i < loops; i++ {
		goGetResults(objetos[lastIndex : lastIndex+BucketSize])
		lastIndex += BucketSize
	}

	if remainder > 0 {
		goGetResults(objetos[lastIndex : lastIndex+remainder])
	}
}
func goGetResults(objetos []string) {
	transport := &http.Transport{DisableCompression: false}
	client := &http.Client{Transport: transport}

	envelope := &EnvelopeRequest{
		XmlnsSoap: "http://schemas.xmlsoap.org/soap/envelope/",
		XmlnsCli:  "http://cliente.bean.master.sigep.bsb.correios.com.br/",
		Header:    EnvelopeHeaderRequest{},
		Body: EnvelopeBodyRequest{
			ConsultaSRO: consultaSRO{
				TipoConsulta:  "L",
				TipoResultado: "U",
				UsuarioSRO:    "ECT",
				SenhaSRO:      "SRO",
				ListaObjetos:  objetos,
			},
		},
	}

	envelopeXML, _ := xml.Marshal(envelope)
	buffer := bytes.NewBuffer(envelopeXML)
	req, err := http.NewRequest("POST", "https://apps.correios.com.br/SigepMasterJPA/AtendeClienteService/AtendeCliente", buffer)
	if err != nil {
		log.Panic("Falha ao criar a requisição: ", err.Error())
	}

	req.Header.Add("Content-Type", "text/xml; charset=UTF-8")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Falha ao executar a consulta: ", err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Erro ao retornar o corpo da requisição", err)
	}

	if resp.StatusCode != 200 {
		//log.Printf("Etiqueta: %s -> Erro %s no acesso ao recurso ou objeto inexistente.", objetos, resp.Status)
		//return
		log.Println("Ocorreu um erro ao consultar os objetos.")
	} else {
		//log.Println(string(data))
	}

	var result envelopeResult

	if err := xml.Unmarshal(data, &result); err != nil {
		log.Println(err)
	}

	var xmlResult rastro

	if err := xml.Unmarshal([]byte(result.XMLBody.Consulta.XMLReturn), &xmlResult); err != nil {
		log.Println(err)
	}
  
	for _, v := range xmlResult.Objetos {
		log.Printf("%s -> %s dia %s às %s em %s", v.Numero, v.Eventos.Descricao, v.Eventos.Data, v.Eventos.Hora, strings.Title(strings.ToLower(v.Eventos.Cidade)))
	}

}

func readFromCsv() []string {
	var objetos []string
	csvFile, _ := os.Open("objetos.csv")
	reader := csv.NewReader(bufio.NewReader(csvFile))
	for {
		line, error := reader.Read()
		if error == io.EOF {
			break
		} else if error != nil {
			log.Fatal(error)
		}
		objetos = append(objetos, line[0])
	}
	return objetos
}
