# Running the demo

The following steps explain how to run the Llama Stack MCP demo. These steps also assume the MCP Server operator is running already and an MCPServer CR has already been reconciled. If these steps have not been fulfilled, please refer to [this](../README.md) before attempting to run the demo.

## Step 1: Set up ollama:
```
curl -fsSL https://ollama.com/install.sh | sh
```

### Pull the llama3.2:3b mode
```
ollama pull llama3.2:3b
```

 Verify that Ollama is running (Completely optional, but recommended)

```
curl http://localhost:11434/
```

## Step 2: Start the llama-stack-server

> **Note:**  
> If you run the `llama-stack-server` container frequently, you may need to delete and recreate the `~/.llama` directory to avoid issues.

```
mkdir -p ~/.llama
docker run -it \
--pull always \
-p 8321:8321 \
-v ~/.llama:/root/.llama \
--network=host \
llamastack/distribution-starter \
--port 8321 \
--env INFERENCE_MODEL=llama3.2:3b \
--env OLLAMA_URL=http://localhost:11434 \
--env ENABLE_OLLAMA=ollama
```

## Step 3: Register the model:

 Install the llama-stack-client CLI if not already installed
```
pip install llama-stack
```

Then, register the llama model with the llama-stack-server
```
llama-stack-client models register llama3.2:3b --provider-id ollama --model-type llm
```

## Step 4: Run the script:

Firstly, set up and activate the python virtual environment
```
python -m venv venv/
source venv/bin/activate
```

Next, install the necessary dependencies to the environment
`pip install -r demo/requirements.txt`


After installing the necessary dependencies, set up the required environment variables.
```
export REMOTE_MCP_URL="http://$(oc get route demo -n mcp-server-operator-system -o jsonpath='{.spec.host}')/sse"
export REMOTE_BASE_URL="http://localhost:8321"
```

Lastly, run the script.
```
REMOTE_BASE_URL=$REMOTE_BASE_URL REMOTE_MCP_URL=$REMOTE_MCP_URL python demo/llama_stack_mcp_agent.py -r -a
```

This command in specific automatically provides the model with the prompt within the demo python script. If you wish to put in a custom prompt, simply run this command instead:
```
REMOTE_BASE_URL=$REMOTE_BASE_URL REMOTE_MCP_URL=$REMOTE_MCP_URL python demo/llama_stack_mcp_agent.py -r
```