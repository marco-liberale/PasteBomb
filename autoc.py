#!/usr/bin/python3

def parse_comments_file(comments_file_path):
    comments = {}
    with open(comments_file_path, 'r') as file:
        for line in file:
            if ': ' in line:
                line_num, comment = line.split(': ', 1)
                if line_num.isdigit():
                    comments[int(line_num)] = comment.strip()
    return comments

def add_comments_to_code(code_file_path, comments):
    with open(code_file_path, 'r') as file:
        code_lines = file.readlines()

    for line_num, comment in comments.items():
        if line_num - 1 < len(code_lines):
            code_lines[line_num - 1] = comment + '\n' + code_lines[line_num - 1]

    with open(code_file_path, 'w') as file:
        file.writelines(code_lines)

def main():
    comments_file_path = input("Enter the path to the comments file: ")
    code_file_path = input("Enter the path to the code file: ")

    comments = parse_comments_file(comments_file_path)
    add_comments_to_code(code_file_path, comments)

    print("Comments have been added to the code file.")

if __name__ == "__main__":
    main()
