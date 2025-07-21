from llama_stack_client.lib.agents.agent import Agent
from llama_stack_client.lib.agents.event_logger import EventLogger
from llama_stack_client.types.tool_group import McpEndpoint
from llama_stack_client import LlamaStackClient
import argparse
import logging
import os
from dotenv import load_dotenv

load_dotenv()

# Logger
logger = logging.getLogger(__name__)
logger.setLevel(logging.INFO)

# Only add handler if none exist
if not logger.hasHandlers():
    stream_handler = logging.StreamHandler()
    stream_handler.setLevel(logging.INFO)
    formatter = logging.Formatter('%(levelname)s:%(name)s:%(message)s')
    stream_handler.setFormatter(formatter)
    logger.addHandler(stream_handler)

# Runtime Arguments
parser = argparse.ArgumentParser()
parser.add_argument("-r", "--remote", help="Uses the remote_url", action="store_true")
parser.add_argument("-s", "--session-info-on-exit", help="Prints agent session info on exit", action="store_true")
parser.add_argument("-a", "--auto", help="Automatically runs examples, and does not start a chat session", action="store_true")
args = parser.parse_args()

# Model
model="llama3.2:3b"

# Connect to a llama stack server
if args.remote:
    base_url = os.getenv("REMOTE_BASE_URL")
    mcp_url = os.getenv("REMOTE_MCP_URL")
else:
    base_url="http://localhost:8321"
    mcp_url="http://host.containers.internal:8000/sse"

client = LlamaStackClient(base_url=base_url)
logger.info(f"Connected to Llama Stack server @ {base_url} \n")

# Get tool info and register tools
registered_tools = client.tools.list()
registered_toolgroups = [t.toolgroup_id for t in registered_tools]

if "mcp::openshift" not in registered_toolgroups:
    # Register MCP tools
    try:
        client.toolgroups.register(
            toolgroup_id="mcp::openshift",
            provider_id="model-context-protocol",
            mcp_endpoint=McpEndpoint(uri=mcp_url),
        )
        logger.info("MCP tools registered successfully")
    except Exception as e:
        logger.error(f"Error registering MCP tools: {e}")
        exit(1)

logger.info(f"""Your Server has access the the following toolgroups:
{set(registered_toolgroups)}
""")

# Create simple agent with tools
agent = Agent(
    client,
    model=model,
    instructions = """You are a helpful assistant. You have access to a number of tools.
    Whenever a tool is called, be sure return the Response in a friendly and helpful tone.
    When you are asked to search the web you must use a tool.
    """ ,
    tools=["mcp::openshift"],
    tool_config={"tool_choice":"auto"}
)

if args.auto:
    user_prompts = ["""Get the pods in the namespace mcp-server-operator-system without using the labelSelector and print just their names"""]
    session_id = agent.create_session(session_name="Auto_demo")

    for i, prompt in enumerate(user_prompts):
        turn_response = agent.create_turn(
            messages=[
                {
                    "role":"user",
                    "content": prompt
                }
            ],
            session_id=session_id,
            stream=True,
        )
        for log in EventLogger().log(turn_response):
            log.print()
else:
    # Start a chat session
    session_id = agent.create_session(session_name="Conversation_demo")
    logger.info("Chat session started. Type '/bye' to exit.")

    while True:
        user_input = input(">>> ")
        if "/bye" in user_input:
            if args.session_info_on_exit:
                agent_session = client.agents.session.retrieve(session_id=session_id, agent_id=agent.agent_id)
                print( agent_session.to_dict())
            break

        turn_response = agent.create_turn(
            session_id=session_id,
            messages=[{"role": "user", "content": user_input}],
        )

        for log in EventLogger().log(turn_response):
            log.print()