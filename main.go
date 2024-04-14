package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

func main() {
	words, err := readWordsFromFile("words.txt")
	if err != nil {
		fmt.Println("Ошибка чтения слов из файла:", err)
		return
	}

	var wg sync.WaitGroup

	results := make(chan string)

	for _, word := range words {
		wg.Add(1)
		go func(word string) {
			defer wg.Done()
			response, err := makeRequest(word, 0)
			if err != nil {
				fmt.Println("Ошибка при отправке запроса для слова", word, ":", err)
				return
			}
			results <- response

			response2, err := makeRequest(word, 20)
			if err != nil {
				fmt.Println("Ошибка при отправке второго запроса для слова", word, ":", err)
				return
			}
			results <- response2
		}(word)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for response := range results {
		fmt.Println(response)
	}
}

func readWordsFromFile(filename string) ([]string, error) {
	var words []string

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		words = append(words, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return words, nil
}

func makeRequest(word string, from int) (string, error) {
	data := url.Values{}
	data.Set("xmlData", fmt.Sprintf(`<elasticrequest><personid>0</personid><buid>0</buid><filters><mainSearchBar><value>%s</value><type>phrase_prefix</type><minimum_should_match>100%%</minimum_should_match></mainSearchBar><purchAmount><minvalue></minvalue><maxvalue></maxvalue></purchAmount><PublicDate><minvalue></minvalue><maxvalue></maxvalue></PublicDate><PurchaseStageTerm><value></value><visiblepart></visiblepart></PurchaseStageTerm><SourceTerm><value></value><visiblepart></visiblepart></SourceTerm><RegionNameTerm><value></value><visiblepart></visiblepart></RegionNameTerm><RequestStartDate><minvalue></minvalue><maxvalue></maxvalue></RequestStartDate><RequestDate><minvalue></minvalue><maxvalue></maxvalue></RequestDate><AuctionBeginDate><minvalue></minvalue><maxvalue></maxvalue></AuctionBeginDate><okdp2MultiMatch><value></value></okdp2MultiMatch><okdp2tree><value></value><productField></productField><branchField></branchField></okdp2tree><classifier><visiblepart></visiblepart></classifier><orgCondition><value></value></orgCondition><orgDictionary><value></value></orgDictionary><organizator><visiblepart></visiblepart></organizator><CustomerCondition><value></value></CustomerCondition><CustomerDictionary><value></value></CustomerDictionary><customer><visiblepart></visiblepart></customer><PurchaseWayTerm><value></value><visiblepart></visiblepart></PurchaseWayTerm><PurchaseTypeNameTerm><value></value><visiblepart></visiblepart></PurchaseTypeNameTerm><BranchNameTerm><value></value><visiblepart></visiblepart></BranchNameTerm><isSharedTerm><value></value><visiblepart></visiblepart></isSharedTerm><isHasComplaint><value></value></isHasComplaint><notificationFeatures><value></value><visiblepart></visiblepart></notificationFeatures><statistic><totalProc></totalProc><TotalSum></TotalSum><DistinctOrgs></DistinctOrgs></statistic></filters><fields><field>TradeSectionId</field><field>purchAmount</field><field>purchCurrency</field><field>purchCodeTerm</field><field>PurchaseTypeName</field><field>purchStateName</field><field>BidStatusName</field><field>OrgName</field><field>SourceTerm</field><field>PublicDate</field><field>RequestDate</field><field>RequestStartDate</field><field>RequestAcceptDate</field><field>EndDate</field><field>CreateRequestHrefTerm</field><field>CreateRequestAlowed</field><field>purchName</field><field>BidName</field><field>SourceHrefTerm</field><field>objectHrefTerm</field><field>needPayment</field><field>IsSMP</field><field>isIncrease</field><field>isHasComplaint</field><field>purchType</field></fields><sort><value>default</value><direction></direction></sort><aggregations><empty><filterType>filter_aggregation</filterType><field></field></empty></aggregations><size>20</size><from>%d</from></elasticrequest>`, word, from))
	data.Set("orgId", "0")
	data.Set("targetPageCode", "UnitedPurchaseList")
	data.Set("PID", "0")

	reqBody := strings.NewReader(data.Encode())

	req, err := http.NewRequest("POST", "https://www.sberbank-ast.ru/SearchQuery.aspx?name=Main", reqBody)
	if err != nil {
		return "", err
	}

	req.Header.Add("Accept", "*/*")
	req.Header.Add("Accept-Encoding", "gzip, deflate, br")
	req.Header.Add("Accept-Language", "ru")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Content-Length", fmt.Sprintf("%d", reqBody.Size()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Add("Cookie", `es_nonathorized_last_filters|UnitedPurchaseList=...; BotMitigationCookie_11412016071036030678="..."; _ym_isad=2; _ym_d=1712597958; _ym_uid=1712597958227773013; ASP.NET_SessionId=ddhyw0clompabpj4yrx5oc1i`) // Усечено для краткости
	req.Header.Add("Host", "www.sberbank-ast.ru")
	req.Header.Add("Origin", "https://www.sberbank-ast.ru")
	req.Header.Add("Referer", "https://www.sberbank-ast.ru/UnitedPurchaseList.aspx")
	req.Header.Add("Sec-Fetch-Dest", "empty")
	req.Header.Add("Sec-Fetch-Mode", "cors")
	req.Header.Add("Sec-Fetch-Site", "same-origin")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15")
	req.Header.Add("X-Requested-With", "XMLHttpRequest")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var bodyContent string
	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return "", err
		}
		defer reader.Close()

		body, err := ioutil.ReadAll(reader)
		if err != nil {
			return "", err
		}

		bodyContent = string(body)
	} else {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		bodyContent = string(body)
	}

	return fmt.Sprintf("Статус ответа для слова '%s' (from=%d): %s\nТело ответа: %s", word, from, resp.Status, bodyContent), nil
}
