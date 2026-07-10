/* ================================================================
   wc-render.js — World Cup 2026 Tournament Page Renderer
   Reads window.WC_DATA and dynamically renders all page sections.
   Exports functions for the HTML to call after WC_DATA is loaded.
   ================================================================ */

/* ─── Helpers ──────────────────────────────────────────────── */
function teamColor(name) {
  if (!name || name === 'TBD' || name.indexOf('/') !== -1) return '#a0aec0';
  var c = WC_DATA.teams[name];
  return c ? c : '#666';
}

function teamInitials(name) {
  if (!name || name === 'TBD') return '?';
  var clean = name.replace(/[^a-zA-Z ]/g, '');
  var parts = clean.split(/\s+/);
  if (parts.length >= 2) return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
  return name.substring(0, 2).toUpperCase();
}

function el(tag, className) {
  var e = document.createElement(tag);
  if (className) e.className = className;
  return e;
}

/* ─── Match Card Renderer ──────────────────────────────────── */
export function matchCard(m, headerCls, headerText) {
  var card = el('div', 'match-card');
  var head = el('div', 'match-card-header ' + headerCls);
  head.textContent = headerText;

  var body = el('div', 'match-card-body');
  var homeWin = m.winner === m.home;

  // Home team row (circle left, name right)
  var r1 = el('div', 'match-row' + (homeWin ? ' winner' : ''));
  var hc = el('span', 'team-circle small'); hc.style.background = teamColor(m.home); hc.textContent = teamInitials(m.home);
  var hn = el('span', 'team-name'); hn.textContent = m.home;
  r1.appendChild(hc); r1.appendChild(document.createTextNode(' ')); r1.appendChild(hn);
  body.appendChild(r1);

  // Score / status
  var scRow = el('div', ''); scRow.style.cssText = 'text-align:center;padding:0.3rem 0';
  if (m.hs !== null && m.hs !== undefined) {
    var s = el('span', 'score'); s.textContent = m.hs + ' – ' + m.as;
    scRow.appendChild(s);
  } else {
    var up = el('span', 'status upcoming'); up.textContent = 'Upcoming';
    scRow.appendChild(up);
  }
  body.appendChild(scRow);

  // Away team row (name left, circle right)
  var r2 = el('div', 'match-row match-row-away');
  var an = el('span', 'team-name'); an.textContent = m.away;
  var ac = el('span', 'team-circle small'); ac.style.background = teamColor(m.away); ac.textContent = teamInitials(m.away);
  r2.appendChild(an); r2.appendChild(document.createTextNode(' ')); r2.appendChild(ac);
  body.appendChild(r2);

  // Extra info: status badges, scorers, pens, aet
  var extra = el('div', ''); extra.style.cssText = 'text-align:center';
  if (m.winner) {
    var st = el('span', 'status win'); st.textContent = '★ ' + m.winner + ' advances';
    extra.appendChild(st);
  }
  if (m.aet) {
    var aetEl = el('span', 'status aet'); aetEl.textContent = 'AET';
    extra.appendChild(aetEl); extra.appendChild(document.createTextNode(' '));
  }
  if (m.pens) {
    var pe = el('span', 'status pens'); pe.textContent = m.pens + ' on pens';
    extra.appendChild(pe); extra.appendChild(document.createTextNode(' '));
  }

  // Scorers
  if (m.scorers && m.scorers.length > 0) {
    for (var si = 0; si < m.scorers.length; si++) {
      var sg = el('div', 'penalty-note');
      sg.textContent = '⚽ ' + m.scorers[si].n + ' (' + m.scorers[si].m + ')';
      extra.appendChild(sg);
    }
  }
  body.appendChild(extra);

  // Venue
  if (m.venue) {
    var vc = el('div', ''); vc.style.cssText = 'font-size:0.68rem;color:var(--gray-400);text-align:center;margin-top:0.25rem';
    vc.textContent = '📍 ' + m.venue;
    body.appendChild(vc);
  }

  card.appendChild(head);
  card.appendChild(body);
  return card;
}

/* ─── Bracket Match Renderer ───────────────────────────────── */
function bracketMatch(m) {
  var card = el('div', 'bm-card' + (m.played ? ' qf-played' : ''));

  if (m.played && m.winner) {
    [
      { n: m.home, s: m.hs, w: m.winner === m.home },
      { n: m.away, s: m.as, w: m.winner === m.away }
    ].forEach(function(t) {
      var row = el('div', 'bm-team-row' + (t.w ? ' winner' : ''));
      var c = el('span', 'bm-circle'); c.style.background = teamColor(t.n); c.textContent = teamInitials(t.n);
      var nm = el('span', 'bm-name'); nm.textContent = t.n;
      var sc = el('span', 'bm-score'); sc.textContent = (t.s !== null && t.s !== undefined) ? t.s : '';
      row.appendChild(c); row.appendChild(nm); row.appendChild(sc);
      card.appendChild(row);
    });
  } else {
    [m.home, m.away].forEach(function(t) {
      var row = el('div', 'bm-team-row');
      var c = el('span', 'bm-circle'); c.style.background = teamColor(t); c.textContent = teamInitials(t);
      var nm = el('span', 'bm-name'); nm.textContent = t;
      row.appendChild(c); row.appendChild(nm);
      card.appendChild(row);
    });
  }
  return card;
}

/* ─── Render: Featured QF Card ─────────────────────────────── */
export function renderFeaturedQF() {
  var container = document.getElementById('featuredQF');
  if (!container) return;

  var m = WC_DATA.knockout.qf[0];
  if (!m || !m.winner) return;

  var html = '<div class="featured-qf">' +
    '<div class="featured-label">★ Quarter-final 1 — Played ' + m.date + '</div>' +
    '<div class="featured-matchup">' +
      '<div class="featured-team">' +
        '<span class="ft-circle" style="background:' + teamColor(m.home) + '">' + teamInitials(m.home) + '</span>' +
        '<div class="ft-name">' + m.home + '</div>' +
      '</div>' +
      '<div class="featured-vs">' + m.hs + ' – ' + m.as + '</div>' +
      '<div class="featured-team">' +
        '<span class="ft-circle" style="background:' + teamColor(m.away) + '">' + teamInitials(m.away) + '</span>' +
        '<div class="ft-name">' + m.away + '</div>' +
      '</div>' +
    '</div>';

  if (m.scorers && m.scorers.length > 0) {
    html += '<div class="featured-score">';
    for (var si = 0; si < m.scorers.length; si++) {
      html += '<span style="margin:0 0.5rem;font-size:0.85rem;opacity:0.9">⚽ ' + m.scorers[si].n + ' ' + m.scorers[si].m + '</span>';
    }
    html += '</div>';
  }

  html += '<div class="featured-venue">📍 ' + m.venue + ' · ' + m.winner + ' advance to Semi-final</div>' +
  '</div>';

  container.innerHTML = html;
}

/* ─── Render: Quarter-finals ───────────────────────────────── */
export function renderQFAll() {
  var grid = document.getElementById('qfAll');
  if (!grid) return;
  var qfs = WC_DATA.knockout.qf;
  for (var i = 0; i < qfs.length; i++) {
    var m = qfs[i];
    grid.appendChild(matchCard(m, 'qf', 'Q' + (i + 1) + ' — ' + m.date));
  }
}

/* ─── Render: Full Bracket Tree ────────────────────────────── */
export function renderBracket() {
  var container = document.getElementById('fullBracket');
  if (!container) return;

  var qf = WC_DATA.knockout.qf;
  var sf = WC_DATA.knockout.sf || [];

  var html = '<div class="bracket-round qf"><div class="bracket-round-header">Quarter-finals</div><div class="bracket-match-list">';
  for (var i = 0; i < qf.length; i++) {
    var m = qf[i];
    html += bracketMatch({ home: m.home, away: m.away, hs: m.hs, as: m.as, winner: m.winner, played: !!m.winner }).outerHTML;
  }
  html += '</div></div>';

  html += '<div class="bracket-spacer"></div>';

  html += '<div class="bracket-round sf"><div class="bracket-round-header">Semi-finals</div><div class="bracket-match-list">';
  for (var si = 0; si < sf.length; si++) {
    var s = sf[si];
    var wQF1 = qf[(s.winnerQF && s.winnerQF[0] !== undefined ? s.winnerQF[0] : 0)];
    var wQF2 = qf[(s.winnerQF && s.winnerQF[1] !== undefined ? s.winnerQF[1] : 1)];
    var t1 = (wQF1 && wQF1.winner) || '?';
    var t2 = (wQF2 && wQF2.winner) || '?';

    var card = el('div', 'bm-card sf-played');
    [
      { n: t1, w: true },
      { n: t2, w: false }
    ].forEach(function(t) {
      var row = el('div', 'bm-team-row');
      if (t.n === '?') { row.style.opacity = '0.4'; row.style.fontStyle = 'italic'; }
      var c = el('span', 'bm-circle'); c.style.background = teamColor(t.n);
      c.textContent = (t.n.length <= 2) ? t.n : teamInitials(t.n);
      var nm = el('span', 'bm-name'); nm.textContent = t.n;
      row.appendChild(c); row.appendChild(nm);
      card.appendChild(row);
    });
    html += card.outerHTML;
  }
  html += '</div></div>';

  html += '<div class="bracket-spacer"></div>';

  html += '<div class="bracket-round f"><div class="bracket-round-header">Final</div><div class="bracket-match-list">';
  var finalCard = el('div', 'bm-card sf-played');
  finalCard.innerHTML = '<div class="bm-team-row" style="opacity:0.4;font-style:italic"><span class="bm-name" style="text-align:center;width:100%">🏆 Final — ' + (WC_DATA.knockout.final ? WC_DATA.knockout.final.date : '') + '<br>' + (WC_DATA.knockout.final ? WC_DATA.knockout.final.venue : '') + '</span></div>';
  html += finalCard.outerHTML;
  html += '</div></div>';

  container.innerHTML = html;
}

/* ─── Render: Round of 16 ──────────────────────────────────── */
export function renderR16() {
  var grid = document.getElementById('r16Results');
  if (!grid) return;
  var matches = WC_DATA.knockout.r16 || [];
  for (var i = 0; i < matches.length; i++) {
    grid.appendChild(matchCard(matches[i], 'r16', 'R16 — ' + matches[i].date));
  }
}

/* ─── Render: Round of 32 ──────────────────────────────────── */
export function renderR32() {
  var grid = document.getElementById('r32Results');
  if (!grid) return;
  var matches = WC_DATA.knockout.r32 || [];
  for (var i = 0; i < matches.length; i++) {
    grid.appendChild(matchCard(matches[i], 'r32', 'R32 — ' + matches[i].date));
  }
}

/* ─── Render: Group Stage Results ──────────────────────────── */
export function renderGroupStage() {
  var grid = document.getElementById('groupResults');
  if (!grid) return;

  var byDate = {};
  var matches = WC_DATA.groupMatches || [];
  for (var i = 0; i < matches.length; i++) {
    var d = matches[i].date;
    if (!byDate[d]) byDate[d] = [];
    byDate[d].push(matches[i]);
  }

  var dates = [];
  var clsMap = { 'A': 'g-a', 'B': 'g-b', 'C': 'g-c', 'D': 'g-d', 'E': 'g-e', 'F': 'g-f',
                 'G': 'g-7', 'H': 'g-8', 'I': 'g-9', 'J': 'g-10', 'K': 'g-11', 'L': 'g-12' };
  for (var k in byDate) { if (byDate.hasOwnProperty(k)) dates.push(k); }
  dates.sort();

  for (var di = 0; di < dates.length; di++) {
    var dayMatches = byDate[dates[di]];
    var byGroup = {};
    for (var mi = 0; mi < dayMatches.length; mi++) {
      var g = dayMatches[mi].group;
      if (!byGroup[g]) byGroup[g] = [];
      byGroup[g].push(dayMatches[mi]);
    }

    var gkeys = []; for (var gk in byGroup) { if (byGroup.hasOwnProperty(gk)) gkeys.push(gk); }
    gkeys.sort();

    for (var gi = 0; gi < gkeys.length; gi++) {
      var g = gkeys[gi];
      var card = el('div', 'match-card');
      var head = el('div', 'match-card-header ' + clsMap[g]);
      head.textContent = 'Group ' + g + ' — ' + dates[di];
      var body = el('div', 'match-card-body');

      for (var j = 0; j < byGroup[g].length; j++) {
        var m = byGroup[g][j];

        // Home team row
        var r1 = el('div', 'match-row');
        var hc = el('span', 'team-circle small'); hc.style.background = teamColor(m.home); hc.textContent = teamInitials(m.home);
        var hn = el('span', 'team-name'); hn.textContent = m.home;
        r1.appendChild(hc); r1.appendChild(document.createTextNode(' ')); r1.appendChild(hn);
        body.appendChild(r1);

        // Score
        var sc = el('div', ''); sc.style.cssText = 'text-align:center;padding:0.3rem 0';
        var s = el('span', 'score'); s.textContent = m.hs + ' – ' + m.as;
        sc.appendChild(s); body.appendChild(sc);

        // Away team row
        var r2 = el('div', 'match-row match-row-away');
        var an = el('span', 'team-name'); an.textContent = m.away;
        var ac = el('span', 'team-circle small'); ac.style.background = teamColor(m.away); ac.textContent = teamInitials(m.away);
        r2.appendChild(an); r2.appendChild(document.createTextNode(' ')); r2.appendChild(ac);
        body.appendChild(r2);
      }

      card.appendChild(head);
      card.appendChild(body);
      grid.appendChild(card);
    }
  }
}

/* ─── Render: Group Standings ──────────────────────────────── */
export function renderStandings() {
  var wrapper = document.getElementById('groupsWrapper');
  if (!wrapper) return;
  var groups = WC_DATA.groups || [];

  for (var gi = 0; gi < groups.length; gi++) {
    var g = groups[gi];
    var card = el('div', 'group-card');
    var head = el('div', 'group-card-header ' + g.headerClass);
    var titleEl = el('span', ''); titleEl.textContent = 'Group ' + g.letter;
    var badge = el('span', 'status-badge'); badge.textContent = 'Complete';
    head.appendChild(titleEl); head.appendChild(badge);

    var table = el('table', 'standings-table');
    var thead = el('thead');
    var hr = el('tr');
    ['#', 'Team', 'P', 'W', 'D', 'L', 'GD', 'GF', 'Pts'].forEach(function(h) {
      var th = el('th', ''); th.textContent = h; hr.appendChild(th);
    });
    thead.appendChild(hr); table.appendChild(thead);

    var tbody = el('tbody');
    for (var ti = 0; ti < g.teams.length; ti++) {
      var t = g.teams[ti];
      var tr = el('tr', '');
      if (t.qa) tr.className = 'qualifier';
      else if (t.elim) tr.className = 'eliminated';

      var td1 = el('td', ''); td1.textContent = ti + 1; tr.appendChild(td1);

      var tdTeam = el('td', '');
      var tc = el('div', 'team-cell');
      var c = el('span', 'team-circle small'); c.style.background = teamColor(t.name); c.textContent = teamInitials(t.name);
      var ns = el('span', 'team-name'); ns.textContent = t.name;
      tc.appendChild(c); tc.appendChild(ns);
      if (t.qa) { var qb = el('span', 'q-badge'); qb.textContent = 'Q'; tc.appendChild(qb); }
      tdTeam.appendChild(tc); tr.appendChild(tdTeam);

      ['gp', 'w', 'd', 'l'].forEach(function(f) {
        var td2 = el('td', ''); td2.textContent = t[f]; tr.appendChild(td2);
      });
      var tdGD = el('td', ''); tdGD.textContent = t.gd > 0 ? '+' + t.gd : t.gd; tr.appendChild(tdGD);
      var tdGF = el('td', ''); tdGF.textContent = t.gf; tr.appendChild(tdGF);
      var tdPts = el('td', 'pts'); tdPts.textContent = t.pts; tr.appendChild(tdPts);

      tbody.appendChild(tr);
    }
    table.appendChild(tbody);
    card.appendChild(head);
    card.appendChild(table);
    wrapper.appendChild(card);
  }
}

/* ─── Render: Tournament Stats Bar ─────────────────────────── */
export function renderStatsBar() {
  var bar = document.getElementById('statsBar');
  if (!bar) return;
  var stats = WC_DATA.tournamentStats || [];
  for (var i = 0; i < stats.length; i++) {
    var card = el('div', 'stat-card');
    var val = el('div', 'stat-value'); val.textContent = stats[i].value;
    var lbl = el('div', 'stat-label'); lbl.textContent = stats[i].label;
    card.appendChild(val);
    card.appendChild(lbl);
    bar.appendChild(card);
  }
}

/* ─── Render: Eliminated Teams ─────────────────────────────── */
export function renderEliminated() {
  var elContainer = document.getElementById('eliminatedByRound');
  if (!elContainer) return;

  var elim = WC_DATA.eliminated || {};
  var roundLabels = { r32: 'Round of 32', r16: 'Round of 16' };

  for (var key in roundLabels) {
    var teams = elim[key] || [];
    if (!teams.length) continue;

    var div = el('div', ''); div.style.cssText = 'margin-bottom:1.5rem';
    var h3 = el('h3', '');
    h3.style.cssText = 'font-size:0.9rem;font-weight:700;color:var(--blue-deep);margin-bottom:0.5rem';
    h3.textContent = roundLabels[key] + ' — ' + teams.length + ' eliminated';
    div.appendChild(h3);

    var grid = el('div', 'eliminated-grid');
    for (var i = 0; i < teams.length; i++) {
      var tag = el('span', 'eliminated-tag');
      var c = el('span', 'et-circle'); c.style.background = teamColor(teams[i]); c.textContent = teamInitials(teams[i]);
      tag.appendChild(c);
      tag.appendChild(document.createTextNode(teams[i]));
      grid.appendChild(tag);
    }
    div.appendChild(grid);
    elContainer.appendChild(div);
  }

  // Group stage eliminated
  var groupElimContainer = document.getElementById('groupEliminatedGrid');
  if (groupElimContainer) {
    var gTeams = elim.group || [];
    for (var gi = 0; gi < gTeams.length; gi++) {
      var tag = el('span', 'eliminated-tag');
      var c = el('span', 'et-circle'); c.style.background = teamColor(gTeams[gi]); c.textContent = teamInitials(gTeams[gi]);
      tag.appendChild(c);
      tag.appendChild(document.createTextNode(gTeams[gi]));
      groupElimContainer.appendChild(tag);
    }
  }
}

/* ─── Render: Top Scorers ──────────────────────────────────── */
export function renderScorers() {
  var grid = document.getElementById('scorersGrid');
  if (!grid) return;

  var scorers = WC_DATA.scorers || [];
  for (var i = 0; i < scorers.length; i++) {
    var s = scorers[i];
    var card = el('div', 'scorer-card' + (i === 0 ? ' golden-boot' : ''));

    // Rank
    var rank = el('div', 'scorer-rank');
    if (i <= 1) rank.className += ' gold';
    else if (i === 2) rank.className += ' silver';
    else if (i === 3) rank.className += ' bronze';
    rank.textContent = '#' + (i + 1);

    // Info
    var info = el('div', 'scorer-info');
    var nameEl = el('div', 'scorer-name'); nameEl.textContent = s.name;
    var teamEl = el('div', 'scorer-team'); teamEl.textContent = s.team + ' · ' + s.club;
    info.appendChild(nameEl); info.appendChild(teamEl);

    // Goals
    var goalsDiv = el('div', ''); goalsDiv.style.cssText = 'text-align:right';
    var goalsNum = el('div', 'scorer-goals'); goalsNum.textContent = s.goals;
    var goalsLbl = el('div', 'scorer-goals-label');
    if (i === 0 || i === 1) goalsLbl.textContent = '⚽ Golden Boot';
    goalsDiv.appendChild(goalsNum);
    if (goalsLbl.textContent) goalsDiv.appendChild(goalsLbl);

    card.appendChild(rank);
    card.appendChild(info);
    card.appendChild(goalsDiv);
    grid.appendChild(card);
  }
}

/* ─── Render: Venues ───────────────────────────────────────── */
export function renderVenues() {
  var grid = document.getElementById('venuesGrid');
  if (!grid) return;

  var venues = WC_DATA.venues || [];
  for (var i = 0; i < venues.length; i++) {
    var v = venues[i];
    var card = el('div', 'venue-card');
    var accent = el('div', 'venue-card-accent ' + v.flag);

    var body = el('div', 'venue-card-body');
    var nameEl = el('div', 'venue-name'); nameEl.textContent = v.name;
    var cityEl = el('div', 'venue-city'); cityEl.textContent = v.city + ', ' + v.country;

    var details = el('div', 'venue-details');
    var capEl = el('span', 'venue-detail'); capEl.textContent = '👥 ' + v.capacity;
    var mkEl = el('span', 'venue-detail'); mkEl.textContent = '📋 ' + v.matches;
    details.appendChild(capEl); details.appendChild(mkEl);

    body.appendChild(nameEl); body.appendChild(cityEl); body.appendChild(details);
    card.appendChild(accent); card.appendChild(body);
    grid.appendChild(card);
  }
}

/* ─── Render: Timeline ─────────────────────────────────────── */
export function renderTimeline() {
  var container = document.getElementById('timelineContainer');
  if (!container) return;

  var events = WC_DATA.timeline || [];
  for (var i = 0; i < events.length; i++) {
    var t = events[i];
    var item = el('div', 'timeline-item ' + t.status);

    var dateEl = el('div', 'timeline-date'); dateEl.textContent = t.date;
    var titleEl = el('div', 'timeline-title'); titleEl.textContent = t.title;
    var descEl = el('div', 'timeline-desc'); descEl.textContent = t.desc;

    item.appendChild(dateEl); item.appendChild(titleEl); item.appendChild(descEl);
    container.appendChild(item);
  }
}

/* ─── Render: Upcoming Fixtures ────────────────────────────── */
export function renderUpcoming() {
  var grid = document.getElementById('upcomingMatches');
  if (!grid) return;

  // Remaining QFs (skip first if already played)
  var qfs = WC_DATA.knockout.qf || [];
  for (var i = 1; i < qfs.length; i++) {
    grid.appendChild(matchCard(qfs[i], 'qf', 'Q' + (i + 1) + ' — ' + qfs[i].date));
  }

  // Semi-finals
  var sf = WC_DATA.knockout.sf || [];
  for (var si = 0; si < sf.length; si++) {
    var s = sf[si];
    var card = el('div', 'match-card');
    var head = el('div', 'match-card-header sf');
    head.textContent = (s.id || 'SF') + ' — ' + s.date;

    var body = el('div', 'match-card-body');
    ['TBD', 'TBD'].forEach(function(t) {
      var row = el('div', 'match-row match-row-away');
      var c = el('span', 'team-circle small'); c.style.background = '#a0aec0'; c.textContent = '?';
      var n = el('span', 'team-name'); n.textContent = t;
      row.appendChild(n); row.appendChild(document.createTextNode(' ')); row.appendChild(c);
      body.appendChild(row);
    });
    var sc = el('div', ''); sc.style.cssText = 'text-align:center;padding:0.3rem 0';
    var up = el('span', 'status upcoming'); up.textContent = 'Upcoming';
    sc.appendChild(up); body.appendChild(sc);
    card.appendChild(head); card.appendChild(body);
    grid.appendChild(card);
  }

  // Third place
  var tp = WC_DATA.knockout.thirdPlace;
  if (tp) {
    var tpCard = el('div', 'match-card');
    var tpHead = el('div', 'match-card-header sf'); tpHead.textContent = 'Third Place — ' + tp.date;
    var tpBody = el('div', 'match-card-body');
    ['TBD', 'TBD'].forEach(function(t) {
      var row = el('div', 'match-row match-row-away');
      var c = el('span', 'team-circle small'); c.style.background = '#a0aec0'; c.textContent = '?';
      var n = el('span', 'team-name'); n.textContent = t;
      row.appendChild(n); row.appendChild(document.createTextNode(' ')); row.appendChild(c);
      tpBody.appendChild(row);
    });
    var sc2 = el('div', ''); sc2.style.cssText = 'text-align:center;padding:0.3rem 0';
    var up2 = el('span', 'status upcoming'); up2.textContent = 'Upcoming';
    sc2.appendChild(up2); tpBody.appendChild(sc2);
    var vc2 = el('div', ''); vc2.style.cssText = 'font-size:0.68rem;color:var(--gray-400);text-align:center;margin-top:0.25rem';
    vc2.textContent = '📍 ' + tp.venue;
    tpBody.appendChild(vc2);
    tpCard.appendChild(tpHead); tpCard.appendChild(tpBody);
    grid.appendChild(tpCard);
  }

  // Final
  var fin = WC_DATA.knockout.final;
  if (fin) {
    var finCard = el('div', 'match-card');
    var finHead = el('div', 'match-card-header f'); finHead.textContent = 'Final — ' + fin.date + ' 🏆';
    var finBody = el('div', 'match-card-body');
    ['TBD', 'TBD'].forEach(function(t) {
      var row = el('div', 'match-row match-row-away');
      var c = el('span', 'team-circle small'); c.style.background = '#a0aec0'; c.textContent = '?';
      var n = el('span', 'team-name'); n.textContent = t;
      row.appendChild(n); row.appendChild(document.createTextNode(' ')); row.appendChild(c);
      finBody.appendChild(row);
    });
    var sc3 = el('div', ''); sc3.style.cssText = 'text-align:center;padding:0.3rem 0';
    var up3 = el('span', 'status upcoming'); up3.textContent = 'Upcoming';
    sc3.appendChild(up3); finBody.appendChild(sc3);
    var vc3 = el('div', ''); vc3.style.cssText = 'font-size:0.68rem;color:var(--gray-400);text-align:center;margin-top:0.25rem';
    vc3.textContent = '📍 ' + fin.venue;
    finBody.appendChild(vc3);
    finCard.appendChild(finHead); finCard.appendChild(finBody);
    grid.appendChild(finCard);
  }
}

/* ─── Render: Storylines ───────────────────────────────────── */
export function renderStorylines() {
  var container = document.getElementById('storylinesContainer');
  if (!container) return;

  var stories = WC_DATA.storylines || [];
  for (var i = 0; i < stories.length; i++) {
    var s = stories[i];
    var card = el('div', 'storyline-card');
    if (s.cssClass) card.className += ' ' + s.cssClass;

    var tagCls = 'sl-tag';
    if (s.tag === 'upset') tagCls += ' upset';
    else if (s.tag === 'golden-boot') tagCls += ' golden-boot';
    else if (s.tag === 'magic-moment') tagCls += ' magic-moment';

    var tag = el('span', tagCls);
    var displayTag = s.tag.replace('-', ' ').replace(/\b\w/g, function(c) { return c.toUpperCase(); });
    tag.textContent = displayTag;

    var titleEl = el('div', 'sl-title'); titleEl.textContent = s.title;
    var bodyEl = el('div', 'sl-body'); bodyEl.textContent = s.body;

    card.appendChild(tag);
    card.appendChild(titleEl);
    card.appendChild(bodyEl);
    container.appendChild(card);
  }
}

/* ─── Render: Hero Stats & Progress ────────────────────────── */
export function renderHero() {
  var stats = WC_DATA.stats || {};

  // Update hero numbers
  var statNums = document.querySelectorAll('.hero-stat .num');
  var labels = ['teams', 'matchesPlayed', 'totalGoals', 'matchesLeft', 'teamsLeft'];
  for (var i = 0; i < labels.length && i < statNums.length; i++) {
    statNums[i].textContent = stats[labels[i]] || '';
  }

  // Progress bar
  var progFill = document.querySelector('.progress-fill');
  var progLabel = document.querySelector('.progress-label span:last-child');
  if (progFill) progFill.style.width = (stats.progressPct || 0) + '%';
  if (progLabel) progLabel.textContent = stats.matchesPlayed + ' / ' + stats.totalMatches + ' matches (' + stats.progressPct + '%)';

  // Badge & date
  var badge = document.querySelector('.hero-badge');
  if (badge) badge.textContent = (WC_DATA.meta.badge || 'Live') + ' — ' + (WC_DATA.meta.currentDate || '');
}

/* ─── Render: Countdown ────────────────────────────────────── */
export function renderCountdown() {
  var el = document.getElementById('countdown');
  if (!el) return;

  var finalDate = new Date(WC_DATA.meta.finalDate || '2026-07-19T23:00:00-04:00');
  var now = new Date();
  var diff = finalDate - now;
  if (diff <= 0) {
    el.innerHTML = '🏆 <strong>Final played!</strong>';
    return;
  }

  var days = Math.floor(diff / (1000 * 60 * 60 * 24));
  var hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
  el.innerHTML = '🏆 <span class="cd-num">' + days + '</span> days, <span class="cd-num">' + hours + '</span> hours until the Final at ' + (WC_DATA.meta.finalVenue || 'MetLife Stadium');
}

/* ─── Tab Navigation ───────────────────────────────────────── */
export function initTabs() {
  var nav = document.getElementById('tabNav');
  if (!nav) return;
  var buttons = nav.querySelectorAll('.tab-btn');

  for (var i = 0; i < buttons.length; i++) {
    buttons[i].addEventListener('click', function() {
      // Deactivate all
      for (var j = 0; j < buttons.length; j++) buttons[j].classList.remove('active');
      var contents = document.querySelectorAll('.tab-content');
      for (var j = 0; j < contents.length; j++) contents[j].classList.remove('active');

      // Activate clicked tab
      this.classList.add('active');
      var tabId = 'tab-' + this.getAttribute('data-tab');
      var target = document.getElementById(tabId);
      if (target) target.classList.add('active');
    });
  }
}

/* ─── Render All Sections ──────────────────────────────────── */
export function renderAll() {
  renderHero();
  renderCountdown();
  renderFeaturedQF();
  renderQFAll();
  renderBracket();
  renderR16();
  renderR32();
  renderGroupStage();
  renderStandings();
  renderStatsBar();
  renderEliminated();
  renderScorers();
  renderVenues();
  renderTimeline();
  renderUpcoming();
  renderStorylines();
  initTabs();
}
