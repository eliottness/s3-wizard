import pytest
import time

from .utils import assert_agent_file, create_file


@pytest.mark.usefixtures('handle_server')
@pytest.mark.parametrize('handle_agent', ['tests/data/simple_config.json'], indirect=True)
class TestS3AgentClassBasic:


    def test_simple_file(self, handle_agent):
        ### GIVEN ###
        file_path = 'simple_file.txt'
        content = 'Hello world'

        create_file(file_path, content)

        ### WHEN ###
        time.sleep(2)

        ### THEN ###
        assert_agent_file(handle_agent, file_path, content)


    def test_simple_folder(self, handle_agent):
        ### GIVEN ###
        file_path = 'simple_folder/simple_folder_file.txt'
        content = 'Hello world'

        create_file(file_path, content)

        ### WHEN ###
        time.sleep(2)

        ### THEN ###
        assert_agent_file(handle_agent, file_path, content)


    def test_subfolder(self, handle_agent):
        ### GIVEN ###
        file_path = 'folder/subfolder/subfolder_file.txt'
        content = 'Hello world'

        create_file(file_path, content)

        ### WHEN ###
        time.sleep(2)

        ### THEN ###
        assert_agent_file(handle_agent, file_path, content)


    def test_same_name_files(self, handle_agent):
        ### GIVEN ###
        first_file_path = 'same_file.txt'
        second_file_path = 'folder/same_file.txt'
        first_content = 'Hello world first'
        second_content = 'Hello world second'

        create_file(first_file_path, first_content)
        create_file(second_file_path, second_content)

        ### WHEN ###
        time.sleep(2)

        ### THEN ###
        assert_agent_file(handle_agent, first_file_path, first_content)
        assert_agent_file(handle_agent, second_file_path, second_content)
