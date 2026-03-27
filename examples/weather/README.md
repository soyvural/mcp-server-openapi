# Weather Example

This example demonstrates how to expose the [Open-Meteo](https://open-meteo.com/) Weather API as MCP tools. No API key required — works out of the box.

## Quick Start

1. **Start the MCP server with the Weather spec:**

```bash
mcp-server-openapi --spec examples/weather/weather.yaml
```

2. **Configure Claude Desktop:**

Edit your Claude Desktop configuration file:

- **macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

Add the following configuration:

```json
{
  "mcpServers": {
    "weather": {
      "command": "/path/to/mcp-server-openapi",
      "args": [
        "--spec",
        "/path/to/examples/weather/weather.yaml"
      ]
    }
  }
}
```

3. **Restart Claude Desktop** and verify the tools are available.

## Available Tools

| Tool Name | Method | Endpoint | Description |
|-----------|--------|----------|-------------|
| `getForecast` | GET | `/v1/forecast` | Get current weather and forecast for any location |
| `getElevation` | GET | `/v1/elevation` | Get elevation in meters for given coordinates |

## Example Prompts

Try these prompts in Claude Desktop:

- "What's the weather in Berlin right now?"
- "Give me a 5-day forecast for New York in Fahrenheit"
- "Compare the weather in Tokyo and London today"
- "What's the elevation of Denver, Colorado?"
- "Will it rain in Paris this week?"

## API Details

The [Open-Meteo API](https://open-meteo.com/) is free for non-commercial use and requires no API key. It provides:

- Current weather conditions
- Hourly and daily forecasts (up to 16 days)
- Temperature, wind, precipitation, humidity, and more
- Elevation data
- Support for Celsius/Fahrenheit and multiple wind speed units
