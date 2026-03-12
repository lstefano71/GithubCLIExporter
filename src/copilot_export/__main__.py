"""Entry point for copilot-export CLI."""

import sys
from .cli import build_parser, run


def main() -> None:
    parser = build_parser()
    args = parser.parse_args()
    sys.exit(run(args))


if __name__ == "__main__":
    main()
