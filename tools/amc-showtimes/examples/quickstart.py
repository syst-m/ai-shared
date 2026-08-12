"""Quickstart example — AMC showtime lookup.

Usage:
    python examples/quickstart.py
"""

from amc_showtimes.client import AMCClient
from amc_showtimes.theaters import get_theater, lookup_theater_number
from amc_showtimes.showtimes import filter_by_formats, get_showtimes
from amc_showtimes.movies import get_movie
from amc_showtimes.formats import list_premium_codes, resolve_format


def main() -> None:
    """Demonstrate the AMC showtimes API integration."""
    # Initialize client — reads API key from ~/.openclaw/.amc-api-key
    client = AMCClient()

    print("=== AMC Showtimes Quickstart ===\n")

    # 1. Look up a theater
    theater_num = lookup_theater_number("metreon")
    if theater_num:
        print(f"Found theater: #{theater_num}")
        theater = get_theater(client, theater_num)
        if theater:
            loc = theater.location
            print(f"  {theater.name}")
            print(f"  {loc.street}, {loc.city}, {loc.stateCode} {loc.zip}\n")

    # 2. List premium formats
    print("Premium formats available:")
    for code in list_premium_codes():
        fmt = resolve_format(code)
        if fmt:
            print(f"  - {fmt.name} ({code})")
    print()

    # 3. Get showtimes (would need active API key)
    print("Fetching today's Dolby + IMAX showtimes at Metreon...")
    try:
        from datetime import datetime
        today = datetime.now().strftime("%Y-%m-%d")
        showtimes = get_showtimes(client, theater_num, date=today)
        filtered = filter_by_formats(showtimes, ["dolby", "imax"])
        print(f"  Found {len(filtered)} premium showtimes")
        for st in filtered:
            print(f"  - {st.movieName} @ {st.showDateTimeLocal}")
    except Exception as e:
        print(f"  (API call failed — expected if key is inactive: {e})")

    # 4. Get movie details
    print("\nFetching movie details for Dune...")
    try:
        movie = get_movie(client, 98765)
        if movie:
            print(f"  {movie.title} ({movie.mpaaRating}, {movie.runtime}min)")
            print(f"  Genre: {movie.genre}")
    except Exception as e:
        print(f"  (API call failed: {e})")


if __name__ == "__main__":
    main()
