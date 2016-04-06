# Cassiopeia

## Description

Cassiopeiaは非デベロッパが手軽に使えるようなIoTバックエンドを作る&繋げるCLIツールです。Cassiopeiaは以下で構成します。

- Transit : デバイスからのデータを1日間の期限付きで蓄積します。EdgeTransit(SORACOM Funnel)とCloudTransit(Amazon Kinesis Streams)のセットで構成します。
- Formatter : Transitからデータを取り出し、変換してAnalyzerにPOSTします。
- Analyzer : データを保存し、分析するWeb画面を提供します。Dockerコンテナ上のElasticsearchとKibanaのセットで構成します。

## Usage

1. AWSとSORACOMのアカウント情報を設定
1. `cas setup`でコンポーネントを作成
1. デバイスからFunnelにデータをPOST
1. `cas pull`でFormatterを実行
1. `cas open`でAnalyzerを表示

## Install

1. [Releases](releases/)ページから実行するOSに合った最新版のファイルをダウンロードします。
1. ダウンロードしたファイルをPATHの通ったディレクトリにファイル名`cas`で配置します。
1. ターミナルを開き、`cas help`を実行して動作を確認します。

## Contribution

1. Fork ([https://github.com/takipone/cassiopeia/fork](https://github.com/takipone/cassiopeia/fork))
1. Create a feature branch
1. Commit your changes
1. Rebase your local changes against the master branch
1. Run test suite with the `go test ./...` command and confirm that it passes
1. Run `gofmt -s`
1. Create a new Pull Request

## Author

[@takipone](https://twitter.com/takipone)
