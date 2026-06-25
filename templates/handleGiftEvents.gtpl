<tr>
    <td>{{ .GiftName }}</td>
    <td>{{ .GiftPoint }}</td>
    <td>{{ if eq .GiftFree false }}SG{{else}}Pt{{ end }}</td>
    <td>{{ .Count }}</td>
    <td>{{ .Pt }}</td>
    <td>{{ .Sum }}</td>
    <td>{{ .Nickname }}</td>
    <td>{{ .UserID }}</td>
    <td>{{ .Avatar }}</td>
    <td>{{ .GiftCode }}</td>
    <td>{{ .GiftType }}</td>
    <td>{{ .H }}</td>
    <td>{{ .D }}</td>
    <td>{{ .At }}</td>
    <td>{{ .Ua }}</td>
    <td>{{ .Aft }}</td>
    <td>{{ .CreatedAtDisplay }}</td>
    <td>{{ .Cl }}</td>
    <td><span class="badge">{{ .Type }}</span></td>
</tr>
