"""Click CLI interface for AMC showtimes tool."""

from __future__ import annotations

import sys
from datetime import datetime, timedelta
from pathlib import Path
from typing import Any

import click

from .client import AMCClient, AMCClientError
from .formats import (
    FORMATS,
    FORMAT_BY_CODE,
    list_premium_codes,
    resolve_format,
)
from .movies import get_movie, search_movies
from .showtimes import filter_after_time, filter_by_formats, get_showtimes
from .theaters import KNOWN_THEATERS, lookup_theater_number


@click.group()
@click.version_option(version="0.1.0", prog_name="amc-showtimes")
@click.option(
    "--api-key-file",
    type=click.Path(exists=True),
    default=None,
    help="Path to AMC API key file.",
)
@click.option(
    "--verbose", "-v",
    is_flag=True,
    help="Enable verbose logging.",
)
@click.pass_context
def cli(ctx: click.Context, api_key_file: str | None, verbose: bool) -> None:
    """AMC Theatres showtime lookup tool.

    Find theaters, showtimes, and premium format screenings.
    """
    import logging
    level = logging.DEBUG if verbose else logging.WARNING
    logging.basicConfig(level=level, stream=sys.stderr)

    key_path = Path(api_key_file) if api_key_file else None
    ctx.ensure_object(dict)
    ctx.obj["client"] = AMCClient(key_path=key_path)


# ------------------------------------------------------------------ #
#  theaters                                                           #
# ------------------------------------------------------------------ #


@cli.command("theaters")
@click.option(
    "--search", "-s",
    type=str,
    default=None,
    help="Search theater name.",
)
@click.pass_context
def cmd_theaters(ctx: click.Context, search: str | None) -> None:
    """List known theaters or search by name."""
    client = ctx.obj["client"]

    if search:
        from .theaters import search_theaters as api_search
        results = api_search(client, search)
        if not results:
            click.echo("No theaters found.", err=True)
            return
        for t in results:
            _print_theater(t)
    else:
        click.echo("Known theaters:")
        seen: set[int] = set()
        for alias, number in sorted(KNOWN_THEATERS.items(), key=lambda x: x[1]):
            if number not in seen:
                seen.add(number)
                try:
                    t = client.get_theater(number)
                    if t:
                        from .theaters import _parse_theater
                        theater_obj = _parse_theater(t)
                        if theater_obj:
                            click.echo(f"\n  # {theater_obj.number} — {theater_obj.name}")
                            loc = theater_obj.location
                            click.echo(
                                f"    {loc.street}, {loc.city}, {loc.stateCode} {loc.zip}"
                            )
                except AMCClientError as exc:
                    click.echo(f"\n  #{number} — Error: {exc}", err=True)


# ------------------------------------------------------------------ #
#  showtimes                                                          #
# ------------------------------------------------------------------ #


@cli.command("showtimes")
@click.option(
    "--theater", "-t",
    type=str,
    required=True,
    help="Theater name/alias (e.g., 'metreon', 'mercado').",
)
@click.option(
    "--date", "-d",
    type=str,
    default=None,
    help="Date in YYYY-MM-DD format. Defaults to today.",
)
@click.option(
    "--formats", "-f",
    type=str,
    default=None,
    help="Comma-separated format codes (e.g., 'dolby,imax').",
)
@click.pass_context
def cmd_showtimes(
    ctx: click.Context,
    theater: str,
    date: str | None,
    formats: str | None,
) -> None:
    """Get showtimes for a theater."""
    client = ctx.obj["client"]

    # Resolve theater number
    theater_num = lookup_theater_number(theater)
    if theater_num is None:
        click.echo(f"Unknown theater: {theater}", err=True)
        click.echo("Known aliases: " + ", ".join(sorted(set(KNOWN_THEATERS.keys()))), err=True)
        sys.exit(1)

    # Default to today
    if date is None:
        date = datetime.now().strftime("%Y-%m-%d")

    try:
        showtimes = get_showtimes(client, theater_num, date=date)
    except AMCClientError as exc:
        click.echo(f"Error fetching showtimes: {exc}", err=True)
        sys.exit(1)

    if not showtimes:
        click.echo(f"No showtimes found for theater #{theater_num} on {date}")
        return

    # Apply format filter
    if formats:
        fmt_list = [f.strip() for f in formats.split(",")]
        showtimes = filter_by_formats(showtimes, fmt_list)
        if not showtimes:
            click.echo(
                f"No showtimes matching formats: {formats}", err=True
            )
            return

    # Display
    _display_showtimes(showtimes, date)


# ------------------------------------------------------------------ #
#  movies                                                             #
# ------------------------------------------------------------------ #


@cli.command("movies")
@click.option(
    "--search", "-s",
    type=str,
    default=None,
    help="Search term.",
)
@click.option(
    "--page", "-p",
    type=int,
    default=1,
    help="Page number.",
)
@click.pass_context
def cmd_movies(ctx: click.Context, search: str | None, page: int) -> None:
    """Search or list movies."""
    client = ctx.obj["client"]

    try:
        if search:
            results = search_movies(client, search, page=page)
        else:
            from .movies import list_now_playing
            results = list_now_playing(client, page=page)
    except AMCClientError as exc:
        click.echo(f"Error fetching movies: {exc}", err=True)
        sys.exit(1)

    if not results:
        click.echo("No movies found.")
        return

    for movie in results:
        fmts = ", ".join(movie.attributes) if movie.attributes else "Standard"
        cast_str = f" | Cast: {', '.join(movie.castList[:3])}" if movie.castList else ""
        runtime_h = movie.runtime // 60
        runtime_m = movie.runtime % 60
        runtime_str = f"{runtime_h}h{runtime_m}m" if runtime_h else f"{movie.runtime}m"

        click.echo(f"\n  {movie.title}")
        click.echo(
            f"    {movie.mpaaRating} | {runtime_str} | {movie.genre}"
            f" | Formats: {fmts}{cast_str}"
        )
        if movie.synopsis:
            # Truncate synopsis for display
            syn = movie.synopsis[:200] + "..." if len(movie.synopsis) > 200 else movie.synopsis
            click.echo(f"    {syn}")


# ------------------------------------------------------------------ #
#  query                                                              #
# ------------------------------------------------------------------ #


@cli.command("query")
@click.option(
    "--theater", "-t",
    type=str,
    required=True,
    help="Theater name/alias.",
)
@click.option(
    "--after", "-a",
    type=str,
    default=None,
    help="Only showtimes at or after HH:MM (24h format).",
)
@click.option(
    "--formats", "-f",
    type=str,
    default=None,
    help="Comma-separated format codes.",
)
@click.pass_context
def cmd_query(
    ctx: click.Context,
    theater: str,
    after: str | None,
    formats: str | None,
) -> None:
    """Quick query: what's playing premium format after time X?

    Example: amc-showtimes query -t metreon -a "19:00" -f dolby,imax
    """
    client = ctx.obj["client"]

    theater_num = lookup_theater_number(theater)
    if theater_num is None:
        click.echo(f"Unknown theater: {theater}", err=True)
        sys.exit(1)

    date = datetime.now().strftime("%Y-%m-%d")
    try:
        showtimes = get_showtimes(client, theater_num, date=date)
    except AMCClientError as exc:
        click.echo(f"Error: {exc}", err=True)
        sys.exit(1)

    # Apply filters
    if formats:
        fmt_list = [f.strip() for f in formats.split(",")]
        showtimes = filter_by_formats(showtimes, fmt_list)

    if after:
        try:
            parts = after.split(":")
            hour = int(parts[0])
            minute = int(parts[1]) if len(parts) > 1 else 0
            showtimes = filter_after_time(showtimes, hour, minute)
        except (ValueError, IndexError):
            click.echo(f"Invalid time format: {after} (use HH:MM)", err=True)
            sys.exit(1)

    if not showtimes:
        click.echo("No matching showtimes found.")
        return

    _display_showtimes(showtimes, date)


# ------------------------------------------------------------------ #
#  enrich                                                             #
# ------------------------------------------------------------------ #


@cli.command("enrich")
@click.option(
    "--movie", "-m",
    type=str,
    required=True,
    help="Movie title to look up.",
)
@click.option(
    "--year", "-y",
    type=int,
    default=None,
    help="Release year.",
)
@click.pass_context
def cmd_enrich(ctx: click.Context, movie: str, year: int | None) -> None:
    """Enrich a movie with IMDB and Rotten Tomatoes ratings."""
    from .enrich import enrich_movie as do_enrich

    ratings = do_enrich(movie, year=year)

    if not ratings.imdb_score and ratings.rt_critics_score is None:
        click.echo(
            "No rating data found. Make sure you have an OMDb API key at "
            f"{OMDB_KEY_PATH}",
            err=True,
        )
        click.echo("Get a free key at: http://www.omdbapi.com/apikey.aspx")
        return

    click.echo(f"\n  {movie}" + (f" ({year})" if year else ""))
    if ratings.imdb_score is not None:
        click.echo(f"    IMDB: {ratings.imdb_score}/10")
        if ratings.imdb_id:
            click.echo(f"    IMDB page: https://www.imdb.com/title/{ratings.imdb_id}")
    if ratings.rt_critics_score is not None:
        click.echo(f"    Rotten Tomatoes (Critics): {ratings.rt_critics_score}%")
    if ratings.rt_audience_score is not None:
        click.echo(f"    Rotten Tomatoes (Audience): {ratings.rt_audience_score}%")


# ------------------------------------------------------------------ #
#  formats                                                            #
# ------------------------------------------------------------------ #


@cli.command("formats")
def cmd_formats() -> None:
    """List all supported format codes."""
    click.echo("Supported formats:")
    for fmt in FORMATS:
        premium = "⭐ PREMIUM" if fmt.is_premium else ""
        click.echo(f"  {fmt.code:15s} — {fmt.name:20s} {premium}")


# ------------------------------------------------------------------ #
#  Display helpers                                                    #
# ------------------------------------------------------------------ #


def _display_showtimes(showtimes: list, date: str) -> None:
    """Display a list of showtimes in a readable format."""
    from .models import Showtime

    click.echo(f"\nShowtimes for {date}:")
    click.echo("-" * 60)

    current_movie: str = ""
    for st in showtimes:
        if isinstance(st, dict):
            # Raw dict — just display what we have
            name = st.get("movieName", "Unknown")
            time_str = _format_time_local(
                st.get("showDateTimeLocal", ""),
                st.get("utcOffset", "-07:00"),
            )
            auditorium = st.get("auditorium", "?")
            attrs = st.get("attributes") or []
            fmt_names = [a.get("name", "") for a in attrs if a.get("code")]
            status = _status_text(st)

            if name != current_movie:
                click.echo(f"\n  {name} ({st.get('mpaaRating', '')})")
                current_movie = name
            click.echo(
                f"    {time_str:>8s}  Auditorium {auditorium}"
                f"  {' | '.join(fmt_names) if fmt_names else 'Standard'}"
                f"  {status}"
            )
        elif isinstance(st, Showtime):
            name = st.movieName
            time_str = _format_time_dt(st.showDateTimeLocal)
            auditorium = st.auditorium
            fmt_names = st.format_names
            status = _status_text_showtime(st)

            if name != current_movie:
                click.echo(f"\n  {name} ({st.mpaaRating})")
                current_movie = name
            click.echo(
                f"    {time_str:>8s}  Auditorium {auditorium}"
                f"  {' | '.join(fmt_names) if fmt_names else 'Standard'}"
                f"  {status}"
            )

    click.echo("")


def _format_time_local(dt_str: str, utc_offset: str) -> str:
    """Format a datetime string as a readable time."""
    try:
        from .showtimes import _parse_datetime
        dt = _parse_datetime(dt_str)
        return dt.strftime("%I:%M %p").lstrip("0")
    except Exception:
        return "Unknown"


def _format_time_dt(dt) -> str:
    """Format a datetime object as readable time."""
    try:
        if hasattr(dt, 'hour'):
            hour = dt.hour
            minute = dt.minute
            if hour == 0:
                return f"12:{minute:02d} AM"
            elif hour < 12:
                return f"{hour}:{minute:02d} AM"
            elif hour == 12:
                return f"12:{minute:02d} PM"
            else:
                return f"{hour - 12}:{minute:02d} PM"
    except Exception:
        pass
    return "Unknown"


def _status_text(st: dict) -> str:
    """Format showtime status text from raw dict."""
    if st.get("isCanceled"):
        return "(CANCELED)"
    if st.get("isSoldOut"):
        return "(SOLD OUT)"
    if st.get("isAlmostSoldOut"):
        return "(Almost Sold Out)"
    return ""


def _status_text_showtime(st) -> str:
    """Format showtime status text from Showtime object."""
    if st.isCanceled:
        return "(CANCELED)"
    if st.isSoldOut:
        return "(SOLD OUT)"
    if st.isAlmostSoldOut:
        return "(Almost Sold Out)"
    return ""


def _print_theater(theater) -> None:
    """Print theater info."""
    click.echo(f"  #{theater.number} — {theater.name}")
    loc = theater.location
    click.echo(f"    {loc.street}, {loc.city}, {loc.stateCode} {loc.zip}")


# Import for enrich command
from .enrich import OMDB_KEY_PATH
