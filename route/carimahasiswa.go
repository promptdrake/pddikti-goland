package route

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

type MahasiswaResult struct {
	Name       string `json:"name"`
	NIM        string `json:"nim"`
	University string `json:"university"`
	Program    string `json:"program"`
}

type UniversityDetail struct {
	Kelompok         string  `json:"kelompok"`
	Pembina          string  `json:"pembina"`
	IDSP             string  `json:"id_sp"`
	KodePT           string  `json:"kode_pt"`
	Email            string  `json:"email"`
	NoTel            string  `json:"no_tel"`
	NoFax            string  `json:"no_fax"`
	Website          string  `json:"website"`
	Alamat           string  `json:"alamat"`
	NamaPT           string  `json:"nama_pt"`
	NmSingkat        string  `json:"nm_singkat"`
	KodePos          string  `json:"kode_pos"`
	ProvinsiPT       string  `json:"provinsi_pt"`
	KabKotaPT        string  `json:"kab_kota_pt"`
	KecamatanPT      string  `json:"kecamatan_pt"`
	LintangPT        float64 `json:"lintang_pt"`
	BujurPT          float64 `json:"bujur_pt"`
	TglBerdiriPT     string  `json:"tgl_berdiri_pt"`
	TglSKPendirianSP string  `json:"tgl_sk_pendirian_sp"`
	SKPendirianSP    string  `json:"sk_pendirian_sp"`
	StatusPT         string  `json:"status_pt"`
	AkreditasiPT     string  `json:"akreditasi_pt"`
	StatusAkreditasi string  `json:"status_akreditasi"`
}

type APIResponse struct {
	Success      bool               `json:"success"`
	Message      string             `json:"message"`
	Data         []MahasiswaResult  `json:"data"`
	DetailKampus []UniversityDetail `json:"detail_kampus"`
}
func CariMahasiswa(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		response := APIResponse{
			Success:      false,
			Message:      "name query parameter is required",
			Data:         []MahasiswaResult{},
			DetailKampus: []UniversityDetail{},
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	target := "https://pddikti.kemdiktisaintek.go.id/search/" + url.QueryEscape(name)
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var html string
	err := chromedp.Run(ctx,
		chromedp.EmulateViewport(1920, 1080),
		chromedp.Navigate(target),


		chromedp.WaitReady(`#root`, chromedp.ByQuery),

	
		chromedp.Sleep(2*time.Second),

		chromedp.ActionFunc(func(ctx context.Context) error {

			return chromedp.WaitVisible(`table, .no-results, .empty-state, [class*="result"], [class*="table"]`, chromedp.ByQueryAll).Do(ctx)
		}),

		chromedp.Sleep(3*time.Second),

		chromedp.OuterHTML("html", &html, chromedp.ByQuery),
	)

	if err != nil {
		response := APIResponse{
			Success:      false,
			Message:      fmt.Sprintf("Failed to render upstream page: %v", err),
			Data:         []MahasiswaResult{},
			DetailKampus: []UniversityDetail{},
		}
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(response)
		return
	}


	fmt.Printf("HTML length: %d\n", len(html))
	if len(html) > 1000 {
		fmt.Printf("HTML sample: %s...\n", html[:1000])
	}

	mahasiswaData := extractMahasiswaData(html)

	var universityDetails []UniversityDetail

	if len(mahasiswaData) > 0 {
		firstUniversity := mahasiswaData[0].University
		fmt.Printf("Searching university details for: %s\n", firstUniversity)

		universityDetail := getUniversityDetails(firstUniversity)
		if universityDetail != nil {
			universityDetails = append(universityDetails, *universityDetail)
		}
	}

	response := APIResponse{
		Success:      len(mahasiswaData) > 0,
		Data:         mahasiswaData,
		DetailKampus: universityDetails,
	}

	if len(mahasiswaData) == 0 {
		response.Message = "Tidak ada hasil pencarian pada bagian Mahasiswa"
	} else {
		response.Message = fmt.Sprintf("%d Query", len(mahasiswaData))
	}

	json.NewEncoder(w).Encode(response)
}

func getUniversityDetails(universityName string) *UniversityDetail {
	searchURL := "https://pddikti.kemdiktisaintek.go.id/search/pt/" + url.QueryEscape(universityName)

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var searchHTML string
	err := chromedp.Run(ctx,
		chromedp.EmulateViewport(1920, 1080),
		chromedp.Navigate(searchURL),
		chromedp.WaitReady(`#root`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
		chromedp.OuterHTML("html", &searchHTML, chromedp.ByQuery),
	)

	if err != nil {
		fmt.Printf("Failed to search university: %v\n", err)
		return nil
	}


	token := extractUniversityToken(searchHTML)
	if token == "" {
		fmt.Println("Failed to extract university token")
		return nil
	}

	fmt.Printf("Found university token: %s\n", token)


	return fetchUniversityDetailsFromAPI(token)
}


func extractUniversityToken(html string) string {
	re := regexp.MustCompile(`/detail-pt/([A-Za-z0-9_-]+={0,2})`)
	matches := re.FindStringSubmatch(html)

	if len(matches) >= 2 {
		return matches[1]
	}

	return ""
}


func fetchUniversityDetailsFromAPI(token string) *UniversityDetail {
	apiURL := fmt.Sprintf("https://api-pddikti.kemdiktisaintek.go.id/pt/detail/%s", token)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		fmt.Printf("Failed to create request: %v\n", err)
		return nil
	}


	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,id;q=0.8")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("DNT", "1")
	req.Header.Set("Origin", "https://pddikti.kemdiktisaintek.go.id")
	req.Header.Set("Referer", "https://pddikti.kemdiktisaintek.go.id/")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Mobile Safari/537.36 Edg/140.0.0.0")
	req.Header.Set("X-User-IP", "182.9.1.224")
	req.Header.Set("sec-ch-ua", `"Chromium";v="140", "Not=A?Brand";v="24", "Microsoft Edge";v="140"`)
	req.Header.Set("sec-ch-ua-mobile", "?1")
	req.Header.Set("sec-ch-ua-platform", `"Android"`)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Failed to fetch university details: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("API returned status code: %d\n", resp.StatusCode)
		return nil
	}

	var universityDetail UniversityDetail
	err = json.NewDecoder(resp.Body).Decode(&universityDetail)
	if err != nil {
		fmt.Printf("Failed to decode university details: %v\n", err)
		return nil
	}

	return &universityDetail
}

func extractMahasiswaData(html string) []MahasiswaResult {
	var results []MahasiswaResult

	
	lower := strings.ToLower(html)


	patterns := []string{
		">mahasiswa<",
		"mahasiswa",
		"data mahasiswa",
		"hasil mahasiswa",
	}

	headingIdx := -1
	for _, pattern := range patterns {
		idx := strings.Index(lower, pattern)
		if idx != -1 {
			headingIdx = idx
			break
		}
	}

	if headingIdx == -1 {
		fmt.Println("No mahasiswa heading found")
		return results
	}

	
	searchArea := html[headingIdx:]
	searchAreaLower := strings.ToLower(searchArea)

	tableStart := strings.Index(searchAreaLower, "<table")
	if tableStart == -1 {
		fmt.Println("No table found after mahasiswa heading")
		return extractFromDivResults(searchArea)
	}

	tableEnd := strings.Index(searchAreaLower[tableStart:], "</table>")
	if tableEnd == -1 {
		fmt.Println("No closing table tag found")
		return results
	}

	tableHTML := searchArea[tableStart : tableStart+tableEnd+8] 
	fmt.Printf("Found table HTML length: %d\n", len(tableHTML))


	return extractFromTable(tableHTML)
}

func extractFromTable(tableHTML string) []MahasiswaResult {
	var results []MahasiswaResult

	rowRegex := regexp.MustCompile(`(?s)<tr[^>]*>(.*?)</tr>`)
	cellRegex := regexp.MustCompile(`(?s)<t[dh][^>]*>(.*?)</t[dh]>`)

	rows := rowRegex.FindAllStringSubmatch(tableHTML, -1)

	for i, row := range rows {
		if i == 0 {
			continue
		}

		cells := cellRegex.FindAllStringSubmatch(row[1], -1)
		if len(cells) >= 3 {
			result := MahasiswaResult{
				Name:       cleanHTML(cells[0][1]),
				NIM:        cleanHTML(cells[1][1]),
				University: cleanHTML(cells[2][1]),
			}

			if len(cells) >= 4 {
				result.Program = cleanHTML(cells[3][1])
			}

			if result.Name != "" || result.NIM != "" {
				results = append(results, result)
			}
		}
	}

	return results
}

func extractFromDivResults(html string) []MahasiswaResult {
	var results []MahasiswaResult
	lines := strings.Split(html, "\n")

	var currentResult MahasiswaResult
	for _, line := range lines {
		line = strings.TrimSpace(cleanHTML(line))
		if line == "" {
			continue
		}
		if strings.Contains(strings.ToLower(line), "nama") && currentResult.Name == "" {
			currentResult.Name = extractValue(line)
		} else if strings.Contains(strings.ToLower(line), "nim") && currentResult.NIM == "" {
			currentResult.NIM = extractValue(line)
		} else if strings.Contains(strings.ToLower(line), "perguruan") && currentResult.University == "" {
			currentResult.University = extractValue(line)
		} else if strings.Contains(strings.ToLower(line), "program") && currentResult.Program == "" {
			currentResult.Program = extractValue(line)
		}

		if currentResult.Name != "" && (currentResult.NIM != "" || currentResult.University != "") {
			results = append(results, currentResult)
			currentResult = MahasiswaResult{}
		}
	}


	if currentResult.Name != "" && (currentResult.NIM != "" || currentResult.University != "") {
		results = append(results, currentResult)
	}

	return results
}

func cleanHTML(s string) string {

	re := regexp.MustCompile(`<[^>]*>`)
	cleaned := re.ReplaceAllString(s, "")


	cleaned = strings.ReplaceAll(cleaned, "&amp;", "&")
	cleaned = strings.ReplaceAll(cleaned, "&lt;", "<")
	cleaned = strings.ReplaceAll(cleaned, "&gt;", ">")
	cleaned = strings.ReplaceAll(cleaned, "&quot;", "\"")
	cleaned = strings.ReplaceAll(cleaned, "&#39;", "'")
	cleaned = strings.ReplaceAll(cleaned, "&nbsp;", " ")

	return strings.TrimSpace(cleaned)
}

func extractValue(line string) string {

	if idx := strings.Index(line, ":"); idx != -1 && idx < len(line)-1 {
		return strings.TrimSpace(line[idx+1:])
	}
	return line
}
