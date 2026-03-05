package docs

import "github.com/swaggo/swag"

const swaggerDoc = `{
  "swagger": "2.0",
  "info": {
    "title": "MarketNews API",
    "description": "API for MarketNews services.",
    "version": "1.0.0"
  },
  "basePath": "/",
  "schemes": [
    "http"
  ],
  "paths": {
    "/health": {
      "get": {
        "summary": "Health check",
        "responses": {
          "200": { "description": "OK" }
        }
      }
    },
    "/api/v1/upcoming": {
      "get": {
        "summary": "List upcoming analyses",
        "parameters": [
          { "name": "from", "in": "query", "required": false, "type": "string", "description": "YYYY-MM-DD" },
          { "name": "to", "in": "query", "required": false, "type": "string", "description": "YYYY-MM-DD" }
        ],
        "responses": {
          "200": { "description": "OK", "schema": { "type": "object" } },
          "400": { "description": "Bad Request" },
          "503": { "description": "Service Unavailable" }
        }
      }
    },
    "/api/v1/upcoming/analysis/{event_id}": {
      "get": {
        "summary": "Get upcoming analysis by event ID",
        "parameters": [
          { "name": "event_id", "in": "path", "required": true, "type": "integer", "format": "int64" }
        ],
        "responses": {
          "200": { "description": "OK", "schema": { "type": "object" } },
          "400": { "description": "Bad Request" },
          "404": { "description": "Not Found" },
          "503": { "description": "Service Unavailable" }
        }
      }
    },
    "/api/v1/upcoming/analysis": {
      "post": {
        "summary": "Run upcoming analysis",
        "parameters": [
          {
            "name": "body",
            "in": "body",
            "required": true,
            "schema": {
              "type": "object",
              "required": ["date", "ff_id"],
              "properties": {
                "date": { "type": "string", "description": "YYYY-MM-DD" },
                "ff_id": { "type": "integer", "format": "int64" }
              }
            }
          }
        ],
        "responses": {
          "200": { "description": "OK", "schema": { "type": "object" } },
          "400": { "description": "Bad Request" },
          "404": { "description": "Not Found" },
          "503": { "description": "Service Unavailable" }
        }
      }
    },
    "/api/v1/trade-analysis": {
      "post": {
        "summary": "Create trade analysis",
        "parameters": [
          {
            "name": "body",
            "in": "body",
            "required": true,
            "schema": { "type": "object" }
          }
        ],
        "responses": {
          "200": { "description": "OK", "schema": { "type": "object" } },
          "400": { "description": "Bad Request" },
          "503": { "description": "Service Unavailable" }
        }
      }
    },
    "/api/v1/trade-analysis/{trade_id}": {
      "get": {
        "summary": "Get trade analysis by ID",
        "parameters": [
          { "name": "trade_id", "in": "path", "required": true, "type": "integer", "format": "int64" }
        ],
        "responses": {
          "200": { "description": "OK", "schema": { "type": "object" } },
          "400": { "description": "Bad Request" },
          "404": { "description": "Not Found" },
          "503": { "description": "Service Unavailable" }
        }
      }
    },
    "/api/v1/trade-analysis/analysis/{trade_id}": {
      "get": {
        "summary": "Get trade analysis text by ID",
        "parameters": [
          { "name": "trade_id", "in": "path", "required": true, "type": "integer", "format": "int64" }
        ],
        "responses": {
          "200": { "description": "OK", "schema": { "type": "object" } },
          "400": { "description": "Bad Request" },
          "404": { "description": "Not Found" },
          "503": { "description": "Service Unavailable" }
        }
      }
    }
  }
}`

type swaggerDocProvider struct{}

func (swaggerDocProvider) ReadDoc() string {
	return swaggerDoc
}

func Register() {
	swag.Register(swag.Name, swaggerDocProvider{})
}
