# saboriアプリのバックエンド
サボり共有アプリ「サボリー」のバックエンドをです。(フロントのリポジトリhttps://github.com/YoshijiFujiwara/vue-native-saborie)  
使用した主な技術は、プログラミング言語がGo, DBにグラフデータベースのNeo4jを使用しています。
このプロジェクトでは以下の点にフィーチャーして作成しました。

* JsonWebTokenを使用した認証機構
* REST API
* グラフDBにNeo4jを使用

Neo4jを使用した理由は、同じようなサボり方の投稿どうしを関連付けて、高速に取得する必要があったからです。
MySQLなどのRDSでは、レコード数が多くなるにつれて、複数テーブルをまたぐとデータの取得がおそくなりますが、グラフDBではレコード同士が物理的なエッジで関連していることに注目しました。

## デプロイ
GoのアプリケーションサーバをAWSのEC2で、  
Neo4jのDBサーバーもEC2で運用しています。

## 公開URL
https://play.google.com/store/apps/details?id=com.yoshijiFujiwara.saborie  で、公開しています。
