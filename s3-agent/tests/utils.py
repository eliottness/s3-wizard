import os
import sqlite3
import subprocess
import time


DEBUG = False
NB_TRY = 10
FILESYSTEM_PATH = './tmp'
S3_AGENT_PATH = "./config"


def run_command(cmd, stdout=None, stderr=None, code=None):
    process = subprocess.run(cmd.split(' '), stderr=subprocess.PIPE, stdout=subprocess.PIPE)
    result = process.returncode, process.stdout.decode(), process.stderr.decode()
    assert True if code   is None else result[0] == code,   result
    assert True if stdout is None else result[1] == stdout, result
    assert True if stderr is None else result[2] == stderr, result


def start_agent(config_path, reset_env=True):
    # Reset env if necessary
    if reset_env:
        run_command(f'rm -rf {S3_AGENT_PATH} {FILESYSTEM_PATH}', code=0)

    # Set config then run s3-agent in sync mode
    run_command(f'./s3-agent --config-folder={S3_AGENT_PATH} config import {config_path}', code=0)
    process = subprocess.Popen(f'./s3-agent --config-folder={S3_AGENT_PATH} sync'.split(' '))

    # Wait for our the filesystem to be ready
    nb_try = 0
    while not os.path.exists(FILESYSTEM_PATH) and nb_try < NB_TRY:
        time.sleep(0.1)
        nb_try += 1
    assert os.path.exists(FILESYSTEM_PATH), 'FS could not be mounted by s3-agent'

    connection = sqlite3.connect(os.path.join(S3_AGENT_PATH, 'sqlite.db'))
    return process, connection


def stop_agent(process, connection, reset_env=True):
    connection.close()
    process.send_signal(subprocess.signal.SIGTERM)
    process.wait()

    # Reset env if necessary
    if reset_env:
        run_command(f'rm -rf {S3_AGENT_PATH} {FILESYSTEM_PATH}', code=0)


def get_rule_entry(cursor):
    cursor.execute("SELECT * FROM s3_rule_tables")
    return cursor.fetchone()


def get_node_entry(cursor, filename):
    path = os.path.join(S3_AGENT_PATH[2:], get_rule_entry(cursor)[0], filename)
    print(path)
    cursor.execute(f"SELECT * FROM s3_node_tables WHERE path = '{path}'")
    return cursor.fetchone()


def assert_rclone_file(cursor, filename):
    rule_entry = get_rule_entry(cursor)
    node_entry = get_node_entry(cursor, filename)
    s3_file_path = os.path.join("remote:bucket-test/s3-agent", rule_entry[0], node_entry[3])
    rclone_config_path = os.path.join(S3_AGENT_PATH, 'rclone.conf.tmp')
    cmd = f'./rclone --config {rclone_config_path} lsf {s3_file_path}'
    run_command(cmd, stdout=node_entry[3] + '\n', stderr='', code=0)


def assert_entry_state(cursor, filename, size, Local, server):
    entry = get_node_entry(cursor, filename)
    assert entry is not None
    assert entry[1] == size, entry
    assert entry[2] == Local, entry
    assert entry[4] == server, entry