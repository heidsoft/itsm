<div align="center">

# 🤖 AI-Native ITSM

## エンタープライズ IT サービス管理 | AI First, Not AI After

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Next.js](https://img.shields.io/badge/Next.js-15.5-000000?style=flat&logo=nextdotjs)](https://nextjs.org)
[![TypeScript](https://img.shields.io/badge/TypeScript-6.0-3178C6?style=flat&logo=typescript)](https://typescriptlang.org)
[![License](https://img.shields.io/badge/License-Apache_2.0-yellowgreen?style=flat)](LICENSE)
[![Backend CI](https://github.com/heidsoft/itsm/actions/workflows/backend-ci.yml/badge.svg)](https://github.com/heidsoft/itsm/actions/workflows/backend-ci.yml)
[![Frontend CI](https://github.com/heidsoft/itsm/actions/workflows/frontend-ci.yml/badge.svg)](https://github.com/heidsoft/itsm/actions/workflows/frontend-ci.yml)
[![Stars](https://img.shields.io/github/stars/heidsoft/itsm?style=flat)](https://github.com/heidsoft/itsm/stargazers)

**[简体中文](./README.md)** · **[English](./README.en.md)** · **日本語**

**ITIL プロセス · BPMN ワークフロー · CMDB · AI 意思決定支援 · Apache-2.0**

</div>

## 概要

ITSM は、企業のデジタル業務プロセスを管理するためのオープンソース IT サービス管理プラットフォームです。ServiceNow クラスの中核 ITSM 機能を目標としながら、軽量なプライベート導入と高い拡張性を重視しています。

チケット、インシデント、問題、変更、リリース、サービスリクエスト、サービスカタログ、ナレッジ、SLA、CMDB、BPMN オーケストレーションを提供します。AI は、トリアージ、要約、ナレッジ検索、ワークフロー提案、監査証跡、権限制御されたツール実行に組み込まれています。

現在は v1.6.x の安定性強化収束フェーズです。レガシーコントローラー層の撤退が完了し `handlers/<domain>` 垂直分割に統一され、ステートマシンの並行保護（CAS）やビジネスエラー意味論（500 に対する 409）が実装され、信頼性の高い非同期実行基盤が整備されています。本番導入前に、セキュリティ設定、バックアップと復旧、容量、SSO・組織同期、監視、災害復旧を利用環境に合わせて検証してください。

## 主な機能

- チケット、インシデント、問題、変更、リリース、サービスリクエスト管理
- BPMN プロセス定義、プロセスインスタンス、ユーザータスク、プロセス連携
- CI タイプ、構成アイテム、関連、トポロジー、影響分析を含む CMDB
- サービスカタログ、ナレッジベース、SLA 監視、エスカレーション
- AI トリアージ、要約、RAG、監査記録、決定論的フォールバック
- RBAC、テナント分離、MSP 基盤、組織管理
- Feishu、WeCom、DingTalk、Webhook 向けコネクターライフサイクル基盤

## クイックスタート

### Docker 開発環境

```bash
git clone https://github.com/heidsoft/itsm.git
cd itsm
cp .env.dev.example .env

make dev-start-docker
make dev-status
make dev-health
```

アクセス先：

- フロントエンド：`http://localhost:3000`
- バックエンド：`http://localhost:8090`
- Swagger：`http://localhost:8090/swagger/index.html`

開発環境専用の初期アカウントは `admin / admin123` です。本番環境の管理者パスワードは `.env.prod` の `ADMIN_PASSWORD`（初回起動時に `itsm-init` コンテナが書き込み）で決まります。ドキュメント内のサンプルパスワードを本番で使用しないでください。

停止：

```bash
make dev-stop-docker
```

### ローカル Go / Next.js 開発

```bash
cp .env.dev.example .env
make dev-start-local

# ローカルアプリケーションプロセスを停止
make dev-stop-local
```

## 本番デプロイ

```bash
# .env.prod と初期ランダムシークレットを生成
make prod-init

# .env.prod 内の REQUIRED 項目と既定の認証情報をすべて変更

# 検証、バックアップ、ビルド、起動、ヘルスチェック
make prod-deploy

make prod-status
make prod-health
```

Compose を手動実行する場合は、本番環境ファイルを必ず明示します。

```bash
docker compose --env-file .env.prod -f docker-compose.prod.yml build itsm-backend itsm-frontend
docker compose --env-file .env.prod -f docker-compose.prod.yml up -d
```

本番環境で開発用パスワードを使用しないでください。稼働開始前に TLS、外部バックアップ、ログ保管、監視を設定してください。

## バージョン付きイメージのビルド

```bash
# itsm-backend:v1.2.0 などのローカルタグ
VERSION=v1.2.0 make build-images

# Registry プレフィックス付きタグ
VERSION=v1.2.0 REGISTRY=ghcr.io/heidsoft make build-images

# アプリケーションイメージを個別にビルド
VERSION=v1.2.0 make build-backend
VERSION=v1.2.0 make build-frontend
```

既定ではホストのネイティブプラットフォームを対象にします。クロスプラットフォーム配布では、必要に応じて `BUILDPLATFORM=linux/amd64` などを指定してください。

## 検証

```bash
make verify-scripts
make check-contracts

cd itsm-backend && go test ./...
cd ../itsm-frontend && npm run type-check && npm run build
```

## ドキュメント

- [ドキュメント一覧](./docs/README.md)
- [ビルド・デプロイガイド](./docs/DEPLOYMENT_OPTIMIZATION.md)
- [運用ランブック](./docs/runbooks/production-initialization.md)
- [設定リファレンス](./docs/getting-started/install.md)
- [本番準備プログラム](./docs/delivery/production-readiness-program.md)
- [アップグレードガイド](./UPGRADE.md)
- [コントリビューションガイド](./CONTRIBUTING.md)

## ライセンス

[Apache License 2.0](./LICENSE) の下で提供されています。
