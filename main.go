package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/gin-gonic/gin"
)

type Article struct {
	Content struct {
		Rendered string `json:"rendered"`
	} `json:"content"`
}

func GetArticle(id int) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("http://10.11.12.90:8083/?rest_route=/wp/v2/posts/%d", id), nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var article Article
	err = json.Unmarshal(bodyText, &article)
	if err != nil {
		return "", err
	}
	return article.Content.Rendered, nil
}

func GetMarkdown(html string) (string, error) {
	converter := md.NewConverter("", true, nil)
	return converter.ConvertString(html)
}

func RemoveImages(input string) string {
	return regexp.MustCompile(`!\[.*?\]\((.*?)\)`).ReplaceAllString(input, "")
}

type ExtensionPointRequest struct {
	Point  string `json:"point"`
	Params struct {
		AppID        string                 `json:"app_id"`
		ToolVariable string                 `json:"tool_variable"`
		Inputs       map[string]interface{} `json:"inputs"`
		Query        string                 `json:"query"`
	} `json:"params"`
}

type UserQuery struct {
	Flagged bool                   `json:"flagged"`
	Action  string                 `json:"action"`
	Inputs  map[string]interface{} `json:"inputs"`
	Query   string                 `json:"query"`
}

func main() {
	router := gin.Default()

	router.POST("/new-api-for-dify", func(c *gin.Context) {
		var req ExtensionPointRequest

		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		articleId, ok := req.Params.Inputs["article"]
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing article id"})
			return
		}

		id, err := strconv.Atoi(articleId.(string))
		if err != nil {
			fmt.Println("转换错误:", err)
		} else {
			fmt.Printf("转换成功: %d\n", id)
		}

		html, err := GetArticle(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		markdown, err := GetMarkdown(html)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var userPayload UserQuery
		userPayload.Flagged = true
		userPayload.Action = "overrided"
		userPayload.Inputs = map[string]interface{}{
			"article": RemoveImages(markdown),
		}
		userPayload.Query = req.Params.Query

		c.JSON(http.StatusOK, userPayload)
	})

	router.Run(":8084")
}
