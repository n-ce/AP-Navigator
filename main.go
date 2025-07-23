package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

const (
	apiBaseURL        = "https://acharyaprashant.org/api/v2/content"
	delayPerBook      = 60 * time.Second // 1 minute delay between processing each book
	outputFilePath    = "book.json"
	booksIndexURL     = apiBaseURL + "/index?contentType=6&lf=2&limit=400&offset=0"
	searchAPIURL      = apiBaseURL + "/search"
	xClientTypeHeader = "web"
)

type APIBook struct {
	ID          string `json:"id"`
	Title       struct {
		English string `json:"english"`
	} `json:"title"`
	Description struct {
		English string `json:"english"`
	} `json:"description"`
}

type APIIndexResponse struct {
	Contents struct {
		Data []APIBook `json:"data"`
	} `json:"contents"`
	Total int `json:"total"`
}

type APIChaptersResponse struct {
	Content struct {
		EnumMask struct {
			SubContents map[string]struct {
				Value struct {
					Chapters []struct {
						ID    string `json:"id"`
						Title struct {
							English string `json:"english"`
						} `json:"title"`
					} `json:"chapters"`
				} `json:"value"`
			} `json:"subContents"`
		} `json:"enumMask"`
	} `json:"content"`
}

type APISearchResponseWrapper struct {
	SearchedContents struct {
		Data []struct {
			ID    string `json:"id"`
			Title struct {
				English string `json:"english"`
			} `json:"title"`
			URI string `json:"uri"`
		} `json:"data"`
	} `json:"searchedContents"`
}

type OutputBook struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Contents    []string `json:"contents"`
}

func main() {
	if err := runUpdater(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("book.json updated successfully!")
}

func runUpdater() error {
	client := &http.Client{Timeout: 30 * time.Second}

	fmt.Printf("Fetching book list from %s...\n", booksIndexURL)
	req, err := http.NewRequest("GET", booksIndexURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request for book index: %w", err)
	}
	req.Header.Set("X-Client-Type", xClientTypeHeader)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch book index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch book index, status code: %d", resp.StatusCode)
	}

	var apiIndexRes APIIndexResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiIndexRes); err != nil {
		return fmt.Errorf("failed to decode book index response: %w", err)
	}

	outputBooks := []OutputBook{}

	for i, book := range apiIndexRes.Contents.Data {
		fmt.Printf("Processing book: %s (ID: %s)\n", book.Title.English, book.ID)

		currentBookHasArticle := false
		outputBookContentsTitles := []string{}

		bookContentsURL := fmt.Sprintf("%s/%s?lf=0", apiBaseURL, book.ID)
		req, err = http.NewRequest("GET", bookContentsURL, nil)
		if err != nil {
			fmt.Printf("  Failed to create request for book contents %s: %v\n", book.ID, err)
			continue
		}
		req.Header.Set("X-Client-Type", xClientTypeHeader)
		contentsResp, err := client.Do(req)
		if err != nil {
			fmt.Printf("  Failed to fetch contents for %s: %v\n", book.ID, err)
			continue
		}
		defer contentsResp.Body.Close()

		if contentsResp.StatusCode != http.StatusOK {
			fmt.Printf("  Failed to fetch contents for %s, status code: %d\n", book.ID, contentsResp.StatusCode)
			continue
		}

		var apiChapters APIChaptersResponse
		if err := json.NewDecoder(contentsResp.Body).Decode(&apiChapters); err != nil {
			fmt.Printf("  Failed to decode chapters for %s: %v\n", book.ID, err)
			continue
		}

		chaptersData, ok := apiChapters.Content.EnumMask.SubContents["1"].Value.Chapters
		if !ok {
			fmt.Printf("  No chapters found at expected path for book %s.\n", book.ID)
		} else {
			for chapterIdx, chapter := range chaptersData {
				outputBookContentsTitles = append(outputBookContentsTitles, chapter.Title.English)

				if chapterIdx == 0 {
					fmt.Printf("  Searching for article presence for first chapter: %s\n", chapter.Title.English)

					searchPayload := map[string]interface{}{
						"q":               chapter.Title.English,
						"sft":             false,
						"limitTypes":      []int{1},
						"offset":          "",
						"lf":              2,
						"limit":           5,
						"forceSearchTerm": false,
					}
					searchBody, _ := json.Marshal(searchPayload)

					req, err = http.NewRequest("POST", searchAPIURL, bytes.NewBuffer(searchBody))
					if err != nil {
						fmt.Printf("  Failed to create search request for %s: %v\n", chapter.Title.English, err)
					} else {
						req.Header.Set("X-Client-Type", xClientTypeHeader)
						req.Header.Set("Content-Type", "application/json")
						searchResp, err := client.Do(req)
						if err != nil {
							fmt.Printf("  Search request failed for %s: %v\n", chapter.Title.English, err)
						} else {
							defer searchResp.Body.Close()
							if searchResp.StatusCode != http.StatusOK {
								fmt.Printf("  Search failed for %s, status: %d\n", chapter.Title.English, searchResp.StatusCode)
							} else {
								var searchResults APISearchResponseWrapper
								if err := json.NewDecoder(searchResp.Body).Decode(&searchResults); err != nil {
									fmt.Printf("  Failed to decode search results for %s: %v\n", chapter.Title.English, err)
								} else {
									if len(searchResults.SearchedContents.Data) > 0 {
										currentBookHasArticle = true
										fmt.Printf("    Article presence detected.\n")
									} else {
										fmt.Printf("    No article presence detected.\n")
									}
								}
							}
						}
					}
				}
			}
		}

		if currentBookHasArticle {
			outputBooks = append(outputBooks, OutputBook{
				ID:          book.ID,
				Title:       book.Title.English,
				Description: book.Description.English,
				Contents:    outputBookContentsTitles,
			})
			fmt.Printf("  Book '%s' included in output (has article).\n", book.Title.English)
		} else {
			fmt.Printf("  Book '%s' excluded from output (no article).\n", book.Title.English)
		}

		if i < len(apiIndexRes.Contents.Data)-1 {
			fmt.Printf("  Waiting %s before next book block...\n", delayPerBook)
			time.Sleep(delayPerBook)
		}
	}

	jsonData, err := json.MarshalIndent(outputBooks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON data: %w", err)
	}

	if err := ioutil.WriteFile(outputFilePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON to file %s: %w", outputFilePath, err)
	}

	return nil
}
