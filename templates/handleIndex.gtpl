<!doctype html>
<html lang="ja">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>SHOWROOM Gift Stream</title>
    <script src="https://unpkg.com/htmx.org@1.9.12"></script>
    <script src="https://unpkg.com/htmx.org/dist/ext/sse.js"></script>
    <style>
        :root {
            color-scheme: light;
            --bg: #f7f2ea;
            --panel: rgba(255, 252, 247, 0.94);
            --line: #dccdbd;
            --ink: #231815;
            --muted: #6f6258;
            --accent: #b2452f;
        }
        * { box-sizing: border-box; }
        body {
            margin: 0;
            font-family: "Hiragino Sans", "Noto Sans JP", sans-serif;
            color: var(--ink);
            background:
                radial-gradient(circle at top left, rgba(255,255,255,0.8), transparent 30%),
                linear-gradient(135deg, #efe2d1 0%, #f7f2ea 55%, #ead8c2 100%);
        }
        .wrap {
            max-width: 100%;
            padding: 24px;
        }
        .panel {
            background: var(--panel);
            border: 1px solid rgba(255,255,255,0.8);
            border-radius: 20px;
            box-shadow: 0 24px 80px rgba(69, 42, 19, 0.12);
            overflow: hidden;
            backdrop-filter: blur(10px);
        }
        .header {
            padding: 20px 24px 12px;
            border-bottom: 1px solid var(--line);
        }
        h1 {
            margin: 0;
            font-size: clamp(24px, 4vw, 40px);
            letter-spacing: 0.04em;
        }
        .sub {
            margin-top: 8px;
            color: var(--muted);
            font-size: 14px;
        }
        .table-wrap {
            overflow: auto;
            max-height: calc(100vh - 180px);
        }
        table {
            width: 100%;
            border-collapse: collapse;
            min-width: 1600px;
        }
        thead th {
            position: sticky;
            top: 0;
            background: #f3e6d8;
            z-index: 1;
        }
        th, td {
            border-bottom: 1px solid var(--line);
            padding: 10px 12px;
            text-align: left;
            white-space: nowrap;
            font-size: 13px;
        }
        tbody tr:nth-child(odd) {
            background: rgba(255,255,255,0.7);
        }
        .badge {
            display: inline-block;
            padding: 2px 8px;
            border-radius: 999px;
            background: rgba(178, 69, 47, 0.12);
            color: var(--accent);
            font-weight: 700;
        }
        @media (max-width: 720px) {
            .wrap { padding: 12px; }
            .header { padding: 16px 16px 10px; }
            th, td { padding: 8px 10px; font-size: 12px; }
        }
    </style>
</head>
<body>
    <div class="wrap">
        <section class="panel">
            <div class="header">
                <h1>SHOWROOM Gift Stream</h1>
                <div class="sub">roomid={{ .RoomID }} | WebSocket 受信 goroutine -> 加工ハブ goroutine -> SSE/HTMX 表示</div>
            </div>
            <div class="table-wrap">
                <table>
                    <thead>
                        <tr>
                            <th>gift_name</th>
                            <th>point</th>
                            <th></th>
                            <th>n</th>
                            <th>Pt.</th>
                            <th>Sum.</th>
                            <th>ac</th>
                            <th>u</th>
                            <th>av</th>
                            <th>g</th>
                            <th>gt</th>
                            <th>h</th>
                            <th>d</th>
                            <th>at</th>
                            <th>ua</th>
                            <th>aft</th>
                            <th>created_at</th>
                            <th>cl</th>
                            <th>t</th>
                        </tr>
                    </thead>
                    <tbody id="gift-table-body"
                           hx-ext="sse"
                              sse-connect="/events/gifts?roomid={{ .RoomID }}"
                           sse-swap="gift_update"
                           hx-swap="afterbegin"></tbody>
                </table>
            </div>
        </section>
    </div>
    <script>
        const maxRows = {{ .MaxRows }};

        function pruneRows() {
            const tbody = document.getElementById('gift-table-body');
            if (!tbody) {
                return;
            }
            while (tbody.rows.length > maxRows) {
                tbody.deleteRow(-1);
            }
        }

        document.body.addEventListener('htmx:afterSwap', function (evt) {
            if (!evt.detail.target || evt.detail.target.id !== 'gift-table-body') {
                return;
            }
            pruneRows();
        });

        document.addEventListener('DOMContentLoaded', function () {
            const tbody = document.getElementById('gift-table-body');
            if (!tbody) {
                return;
            }

            const observer = new MutationObserver(function () {
                pruneRows();
            });
            observer.observe(tbody, { childList: true });

            setInterval(pruneRows, 500);
            pruneRows();
        });
    </script>
</body>
</html>
