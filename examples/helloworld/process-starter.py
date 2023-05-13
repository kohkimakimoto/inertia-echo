#!/usr/bin/env python3
# -*- coding: utf-8 -*-
import argparse
import signal
import subprocess
import sys
import textwrap
import threading
import json


def _color(code):
    def inner(text):
        return "\033[%sm%s\033[0m" % (code, text)
    return inner

bold = _color("1")
dim = _color("2")
italic = _color("3")
underline = _color("4")
blinking = _color("5")
red = _color("31")
green = _color("32")
yellow = _color("33")
blue = _color("34")
magenta = _color("35")
cyan = _color("36")
white = _color("37")


def handle_output(process, prefix, color):
    for line in iter(process.stdout.readline, b''):
        print(coloredPrefix(prefix, color) + line.decode(), end='')


def handle_error(process, prefix, color):
    for line in iter(process.stderr.readline, b''):
        print(coloredPrefix(prefix, color) + line.decode(), end='', file=sys.stderr)


def coloredPrefix(s, color):
    if s == "":
        return s

    s = s + " "
    if color == "red":
        return bold(red(s))
    elif color == "green":
        return bold(green(s))
    elif color == "yellow":
        return bold(yellow(s))
    elif color == "blue":
        return bold(blue(s))
    elif color == "magenta":
        return bold(magenta(s))
    elif color == "cyan":
        return bold(cyan(s))
    elif color == "white":
        return bold(white(s))
    else:
        return bold(s)


def run(cmd):
    p = {}
    if cmd.startswith("{"):
        p = json.loads(cmd)
        if "prefix" not in p:
            p["prefix"] = ""
        if "prefixColor" not in p:
            p["prefixColor"] = ""
        if "command" not in p:
            raise Exception("command is not found in json")
    else:
        p["prefix"] = ""
        p["prefixColor"] = ""
        p["command"] = cmd

    process = subprocess.Popen(p["command"], stdout=subprocess.PIPE, stderr=subprocess.PIPE, shell=True)

    output_thread = threading.Thread(target=handle_output, args=(process, p["prefix"], p["prefixColor"]))
    output_thread.start()

    error_thread = threading.Thread(target=handle_error, args=(process, p["prefix"], p["prefixColor"]))
    error_thread.start()

    process.wait()
    output_thread.join()
    error_thread.join()


def sig_handler(signum, frame):
    sys.exit(0)


def start(args):
    commands = args.command

    # handing signals.
    signal.signal(signal.SIGTERM, sig_handler)
    signal.signal(signal.SIGINT, sig_handler)

    threads = []
    for c in commands:
        t = threading.Thread(target=run, args=(c,))
        threads.append(t)
        t.start()

    # wait for all run command threads finish
    for t in threads:
        t.join()


def main():
    parser = argparse.ArgumentParser(
        description="process-starter.py is a utility to start multiple processes concurrently",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=textwrap.dedent('''\
            description:
              A utility to start multiple processes concurrently.

            example:
              # start multiple processes concurrently
              process-starter.py "ls -la" "pwd" "echo hello"

              # start multiple processes concurrently with json config
              process-starter.py \\
                '{"command": "php -S localhost:8000", "prefix": "[php 8000]", "prefixColor": "blue"}' \\
                '{"command": "php -S localhost:8001", "prefix": "[php 8001]", "prefixColor": "green"}'

            Copyright (c) Kohki Makimoto <kohki.makimoto@gmail.com>
            The MIT License (MIT)
        '''))

    parser.add_argument("command", nargs="*", help="Commands that you want to run concurrently")

    if len(sys.argv) == 1:
        parser.print_help()
        sys.exit(1)

    args = parser.parse_args()
    start(args)


if __name__ == '__main__':
    main()
