<!doctype html>
<html lang="ja">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>SRWebSocket Admin</title>
    <style>
        :root {
            color-scheme: light;
            --bg: #f4f6f8;
            --panel: #ffffff;
            --line: #d8dee7;
            --ink: #1b2430;
            --muted: #5d6b7d;
            --accent: #1d5cff;
        }
        * { box-sizing: border-box; }
        body {
            margin: 0;
            font-family: "Hiragino Sans", "Noto Sans JP", sans-serif;
            background: linear-gradient(180deg, #f7f9fc 0%, #eef3f9 100%);
            color: var(--ink);
        }
        .wrap {
            max-width: 1200px;
            margin: 0 auto;
            padding: 24px;
        }
        .panel {
            background: var(--panel);
            border: 1px solid var(--line);
            border-radius: 16px;
            box-shadow: 0 16px 40px rgba(20, 30, 50, 0.08);
            overflow: hidden;
        }
        .header {
            padding: 18px 20px;
            border-bottom: 1px solid var(--line);
        }
        h1 { margin: 0; font-size: 28px; }
        .sub { margin-top: 6px; color: var(--muted); font-size: 14px; }
        .toolbar {
            padding: 16px 20px;
            border-bottom: 1px solid var(--line);
            background: #fbfcfe;
        }
        .form-row {
            display: flex;
            gap: 12px;
            align-items: center;
            flex-wrap: wrap;
        }
        .form-row input[type="number"] {
            width: 160px;
            padding: 8px 10px;
            border: 1px solid var(--line);
            border-radius: 8px;
            font-size: 14px;
        }
        .form-row label {
            display: inline-flex;
            gap: 6px;
            align-items: center;
            font-size: 14px;
        }
        .form-row button {
            padding: 8px 14px;
            border: 0;
            border-radius: 8px;
            background: var(--accent);
            color: white;
            font-weight: 700;
            cursor: pointer;
        }
        .toolbar a {
            color: var(--accent);
            text-decoration: none;
            font-weight: 700;
        }
        table {
            width: 100%;
            border-collapse: collapse;
        }
        th, td {
            border-bottom: 1px solid var(--line);
            padding: 10px 12px;
            text-align: left;
            font-size: 14px;
            white-space: nowrap;
        }
        thead th {
            background: #f7f9fc;
            position: sticky;
            top: 0;
            z-index: 1;
        }
        tbody tr:nth-child(odd) {
            background: #fcfdff;
        }
        .status-ok { color: #0f7b2d; font-weight: 700; }
        .status-stop { color: #9c3d12; font-weight: 700; }
        .status-wait { color: #a56b00; font-weight: 700; }
        .muted { color: var(--muted); }
        .actions button, .actions a {
            margin-right: 8px;
        }
        .actions button {
            border: 1px solid var(--line);
            background: #fff;
            border-radius: 8px;
            padding: 6px 10px;
            cursor: pointer;
        }
        .actions form {
            display: inline-block;
            margin: 0 6px 0 0;
        }
    </style>
</head>
<body>
    <div class="wrap">
        <section class="panel">
            <div class="header">
                <h1>SRWebSocket Admin</h1>
                <div class="sub">room_config と roomManager の現在状態の一覧</div>
            </div>
            <div class="toolbar">
                <a href="/">データ表示画面へ</a>
                <form class="form-row" method="post" action="/admin/rooms" style="margin-top:12px;">
                    <input type="number" name="roomid" min="1" placeholder="ルームID" required>
                    <label><input type="radio" name="mode" value="once" checked> 一回</label>
                    <label><input type="radio" name="mode" value="always"> 常時</label>
                    <label><input type="checkbox" name="save_data"> 保存</label>
                    <button type="submit">追加</button>
                </form>
            </div>
            <div style="overflow:auto; max-height: calc(100vh - 160px);">
                <table>
                    <thead>
                        <tr>
                            <th>ルームID</th>
                            <th>種別</th>
                            <th>保存</th>
                            <th>状況</th>
                            <th>ユーザー数</th>
                            <th>更新日時</th>
                            <th>操作</th>
                        </tr>
                    </thead>
                    <tbody>
                        {{ range .Rooms }}
                        <tr>
                            <td>{{ .RoomID }}</td>
                            <td>{{ if eq .Mode "always" }}常時{{ else }}一回{{ end }}</td>
                            <td>{{ if .SaveData }}あり{{ else }}なし{{ end }}</td>
                            <td class="{{ if eq .Status "データ取得中" }}status-ok{{ else if eq .Status "データ取得待ち" }}status-wait{{ else }}status-stop{{ end }}">{{ .Status }}</td>
                            <td>{{ .Subscribers }}</td>
                            <td class="muted">{{ .UpdatedAt }}</td>
                            <td class="actions">
                                <a href="/?roomid={{ .RoomID }}">表示</a>
                                {{ if .Running }}
                                    {{ if eq .Mode "always" }}
                                    <form method="post" action="/admin/rooms">
                                        <input type="hidden" name="roomid" value="{{ .RoomID }}">
                                        <input type="hidden" name="action" value="stop-always">
                                        <button type="submit">停止して「一回」に変更</button>
                                    </form>
                                    {{ else }}
                                    <form method="post" action="/admin/rooms">
                                        <input type="hidden" name="roomid" value="{{ .RoomID }}">
                                        <input type="hidden" name="action" value="stop">
                                        <button type="submit">停止</button>
                                    </form>
                                    {{ end }}
                                {{ else }}
                                    <form method="post" action="/admin/rooms">
                                        <input type="hidden" name="roomid" value="{{ .RoomID }}">
                                        <input type="hidden" name="action" value="start">
                                        <input type="hidden" name="mode" value="{{ .Mode }}">
                                        <input type="hidden" name="save_data" value="{{ if .SaveData }}on{{ end }}">
                                        <button type="submit">開始</button>
                                    </form>
                                    {{ if eq .Mode "always" }}
                                    <form method="post" action="/admin/rooms">
                                        <input type="hidden" name="roomid" value="{{ .RoomID }}">
                                        <input type="hidden" name="action" value="mode-once">
                                        <button type="submit">「一回」に変更</button>
                                    </form>
                                    {{ end }}
                                    {{ if eq .Mode "once" }}
                                    <form method="post" action="/admin/rooms">
                                        <input type="hidden" name="roomid" value="{{ .RoomID }}">
                                        <input type="hidden" name="action" value="start-always">
                                        <input type="hidden" name="save_data" value="{{ if .SaveData }}on{{ end }}">
                                        <button type="submit">「常時」に変更して開始</button>
                                    </form>
                                    {{ end }}
                                    {{ if eq .Mode "once" }}
                                    <form method="post" action="/admin/rooms">
                                        <input type="hidden" name="roomid" value="{{ .RoomID }}">
                                        <input type="hidden" name="action" value="delete">
                                        <button type="submit">削除</button>
                                    </form>
                                    {{ end }}
                                {{ end }}
                            </td>
                        </tr>
                        {{ end }}
                    </tbody>
                </table>
            </div>
        </section>
    </div>
</body>
</html>