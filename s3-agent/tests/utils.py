import os
import subprocess


S3_AGENT_PATH = "./config"


def run_command(cmd, stdout=None, stderr=None, code=None):
    process = subprocess.run(cmd.split(' '), stderr=subprocess.PIPE, stdout=subprocess.PIPE)
    assert True if code is None else process.returncode == code, process.returncode
    assert True if stdout is None else process.stdout.decode() == stdout, process.stdout.decode()
    assert True if stderr is None else process.stderr.decode() == stderr, process.stderr.decode()


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
