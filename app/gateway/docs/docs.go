// GENERATED BY THE COMMAND ABOVE; DO NOT EDIT
// This file was generated by swaggo/swag

package docs

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/alecthomas/template"
	"github.com/swaggo/swag"
)

var doc = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{.Description}}",
        "title": "{{.Title}}",
        "contact": {},
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/comments": {
            "post": {
                "description": "Create new single comment",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Comments"
                ],
                "summary": "Create new comment",
                "responses": {
                    "201": {
                        "description": "Created",
                        "schema": {
                            "$ref": "#/definitions/comments.Comment"
                        }
                    }
                }
            }
        },
        "/comments/hotel/{hotel_id}": {
            "get": {
                "description": "Get comments list by hotel uuid",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Comments"
                ],
                "summary": "Get comments by hotel id",
                "parameters": [
                    {
                        "type": "string",
                        "description": "hotel uuid",
                        "name": "hotel_id",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/comments.List"
                        }
                    }
                }
            }
        },
        "/comments/{comment_id}": {
            "get": {
                "description": "Get comment by uuid",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Comments"
                ],
                "summary": "Get comment by id",
                "parameters": [
                    {
                        "type": "string",
                        "description": "comment uuid",
                        "name": "comment_id",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/comments.Comment"
                        }
                    }
                }
            },
            "put": {
                "description": "Update comment by uuid",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Comments"
                ],
                "summary": "Update comment by id",
                "parameters": [
                    {
                        "type": "string",
                        "description": "comment uuid",
                        "name": "comment_id",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/comments.Comment"
                        }
                    }
                }
            }
        },
        "/hotels": {
            "get": {
                "description": "Get hotels list with pagination using page and size query parameters",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Hotels"
                ],
                "summary": "Get hotels list new user",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "page number",
                        "name": "page",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "number of elements",
                        "name": "size",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/hotels.ListResult"
                        }
                    }
                }
            },
            "post": {
                "description": "Create new hotel instance",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Hotels"
                ],
                "summary": "Create new hotel",
                "responses": {
                    "201": {
                        "description": "Created",
                        "schema": {
                            "$ref": "#/definitions/hotels.Hotel"
                        }
                    }
                }
            }
        },
        "/hotels/{hotel_id}": {
            "get": {
                "description": "Get single hotel by uuid",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Hotels"
                ],
                "summary": "Get hotel by id",
                "parameters": [
                    {
                        "type": "string",
                        "description": "hotel uuid",
                        "name": "hotel_id",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/hotels.Hotel"
                        }
                    }
                }
            },
            "put": {
                "description": "Update single hotel data",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Hotels"
                ],
                "summary": "Update hotel data",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "Hotel UUID",
                        "name": "hotel_id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/hotels.Hotel"
                        }
                    }
                }
            }
        },
        "/hotels/{id}/image": {
            "put": {
                "description": "Upload hotel logo image",
                "consumes": [
                    "multipart/form-data"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Hotels"
                ],
                "summary": "Upload hotel image",
                "parameters": [
                    {
                        "type": "string",
                        "description": "hotel uuid",
                        "name": "hotel_id",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/hotels.Hotel"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "comments.Comment": {
            "type": "object",
            "required": [
                "message",
                "rating"
            ],
            "properties": {
                "comment_id": {
                    "type": "string"
                },
                "created_at": {
                    "type": "string"
                },
                "hotel_id": {
                    "type": "string"
                },
                "message": {
                    "type": "string"
                },
                "photos": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "rating": {
                    "type": "number"
                },
                "updated_at": {
                    "type": "string"
                },
                "user_id": {
                    "type": "string"
                }
            }
        },
        "comments.CommentFull": {
            "type": "object",
            "properties": {
                "comment_id": {
                    "type": "string"
                },
                "createdAt": {
                    "type": "string"
                },
                "hotel_id": {
                    "type": "string"
                },
                "message": {
                    "type": "string"
                },
                "photos": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "rating": {
                    "type": "number"
                },
                "updatedAt": {
                    "type": "string"
                },
                "user": {
                    "$ref": "#/definitions/users.CommentUser"
                }
            }
        },
        "comments.List": {
            "type": "object",
            "properties": {
                "comments": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/comments.CommentFull"
                    }
                },
                "hasMore": {
                    "type": "boolean"
                },
                "page": {
                    "type": "integer"
                },
                "size": {
                    "type": "integer"
                },
                "totalCount": {
                    "type": "integer"
                },
                "totalPages": {
                    "type": "integer"
                }
            }
        },
        "hotels.Hotel": {
            "type": "object",
            "required": [
                "city",
                "country",
                "description",
                "email",
                "location",
                "name",
                "rating"
            ],
            "properties": {
                "city": {
                    "type": "string"
                },
                "comments_count": {
                    "type": "integer"
                },
                "country": {
                    "type": "string"
                },
                "created_at": {
                    "type": "string"
                },
                "description": {
                    "type": "string"
                },
                "email": {
                    "type": "string"
                },
                "hotel_id": {
                    "type": "string"
                },
                "image": {
                    "type": "string"
                },
                "latitude": {
                    "type": "number"
                },
                "location": {
                    "type": "string"
                },
                "longitude": {
                    "type": "number"
                },
                "name": {
                    "type": "string"
                },
                "photos": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "rating": {
                    "type": "number"
                },
                "updated_at": {
                    "type": "string"
                }
            }
        },
        "hotels.ListResult": {
            "type": "object",
            "properties": {
                "hasMore": {
                    "type": "boolean"
                },
                "hotels": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/hotels.Hotel"
                    }
                },
                "page": {
                    "type": "integer"
                },
                "size": {
                    "type": "integer"
                },
                "totalCount": {
                    "type": "integer"
                },
                "totalPages": {
                    "type": "integer"
                }
            }
        },
        "users.CommentUser": {
            "type": "object",
            "properties": {
                "avatar": {
                    "type": "string"
                },
                "email": {
                    "type": "string"
                },
                "firstName": {
                    "type": "string"
                },
                "lastName": {
                    "type": "string"
                },
                "role": {
                    "type": "string"
                },
                "userId": {
                    "type": "string"
                }
            }
        }
    }
}`

type swaggerInfo struct {
	Version     string
	Host        string
	BasePath    string
	Schemes     []string
	Title       string
	Description string
}

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = swaggerInfo{
	Version:     "",
	Host:        "",
	BasePath:    "",
	Schemes:     []string{},
	Title:       "",
	Description: "",
}

type s struct{}

func (s *s) ReadDoc() string {
	sInfo := SwaggerInfo
	sInfo.Description = strings.Replace(sInfo.Description, "\n", "\\n", -1)

	t, err := template.New("swagger_info").Funcs(template.FuncMap{
		"marshal": func(v interface{}) string {
			a, _ := json.Marshal(v)
			return string(a)
		},
	}).Parse(doc)
	if err != nil {
		return doc
	}

	var tpl bytes.Buffer
	if err := t.Execute(&tpl, sInfo); err != nil {
		return doc
	}

	return tpl.String()
}

func init() {
	swag.Register(swag.Name, &s{})
}