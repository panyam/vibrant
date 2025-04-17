const fs = require("fs");
const path = require("path");
const webpack = require("webpack");
const CopyPlugin = require("copy-webpack-plugin");
const HtmlWebpackPlugin = require("html-webpack-plugin");
const HtmlWebpackTagsPlugin = require("html-webpack-tags-plugin");
const MiniCssExtractPlugin = require("mini-css-extract-plugin");
const { CleanWebpackPlugin } = require("clean-webpack-plugin");
// const uglifyJsPlugin = require("uglifyjs-webpack-plugin");

const SRC_FOLDERS = ["./components"]
const OUTPUT_FOLDERS = ["./templates"]

const components = [
  // "NotationViewer",
  // "NotationEditor",
  //"ConsoleView",
  ["App", 0, "ts"],
];

module.exports = (_env, options) => {
  context: path.resolve(__dirname);
  const isDevelopment = options.mode == "development";
  return webpackConfigs = {
    devtool: "source-map",
    devServer: {
      hot: true,
      serveIndex: true,
    },
    externals: {
      // CodeMirror: 'CodeMirror',
      // 'GL': "GoldenLayout",
      ace: "commonjs ace",
      // ace: 'ace',
    },
    entry: components.reduce(function (map, comp) {
      const compName = comp[0];
      const compFolder = SRC_FOLDERS[comp[1]];
      const compExt = comp[2];
      map[compName] = path.join(__dirname, `${compFolder}/${compName}.${compExt}`);
      return map;
    }, {}),
    module: {
      rules: [
        {
          test: /\.js$/,
          exclude: ["node_modules/", "dist"].map((x) => path.resolve(__dirname, x)),
          use: ["babel-loader"],
        },
        {
          test: /\.ts$/,
          exclude: [path.resolve(__dirname, "node_modules"), path.resolve(__dirname, "dist")],
          include: SRC_FOLDERS.map((x) => path.resolve(__dirname, x)),
          use: [
            {
              loader: "ts-loader",
              options: { configFile: "tsconfig.json" },
            },
          ],
        },
        {
          test: /\.(png|svg|jpg|jpeg|gif)$/i,
          type: "asset/resource",
        },
        {
          test: /\.(woff|woff2|eot|ttf|otf)$/i,
          type: "asset/resource",
        },
      ],
    },
    resolve: {
      extensions: [".js", ".jsx", ".ts", ".tsx", ".scss", ".css", ".png"],
      fallback: {
        "querystring-es3": false,
        assert: false,
        child_process: false,
        crypto: false,
        fs: false,
        http: false,
        https: false,
        net: false,
        os: false,
        path: false,
        querystring: false,
        stream: require.resolve("stream-browserify"),
        buffer: require.resolve("buffer"),
        tls: false,
        url: false,
        util: false,
        zlib: false,
      },
    },
    output: {
      path: path.resolve(__dirname, "./static/js/gen/"),
      publicPath: "/static/js/gen/",
      // filename: "[name]-[hash:8].js",
      filename: "[name].[contenthash].js",
      library: ["notation", "[name]"],
      libraryTarget: "umd",
      umdNamedDefine: true,
      globalObject: "this",
    },
    plugins: [
      new webpack.ProvidePlugin({
        Buffer: ['buffer', 'Buffer'],
      }),
      new CleanWebpackPlugin(),
      new MiniCssExtractPlugin(),
      ...components.map(
        (component) =>
          new HtmlWebpackPlugin({
            chunks: [component[0]],
            // inject: false,
            filename: path.resolve(__dirname, `${OUTPUT_FOLDERS[component[1]]}/gen.${component[0]}.html`),
            // template: path.resolve(__dirname, `${component}.html`),
            templateContent: "",
            minify: { collapseWhitespace: false },
          }),
      ),
      new webpack.HotModuleReplacementPlugin(),
    ],
    optimization: {
      splitChunks: {
        chunks: "all",
      },
    },
  };
};
