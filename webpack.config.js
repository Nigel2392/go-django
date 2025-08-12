const path = require('path');

const tsLoaderConfig = {
    test: /\.ts$/i,
    use: 'ts-loader',
    exclude: '/node_modules/'
}

const tsxLoaderConfig = {
    test: /\.tsx$/i,
    use: 'ts-loader',
    exclude: '/node_modules/'
}

const cssLoaderConfig = {
    test: /\.css$/i,
    use: [
        'style-loader',
        'css-loader',
    ],
    exclude: '/node_modules/'
}

function baseConfig(rules = []) {
    if (rules.length === 0) {
        rules = [tsLoaderConfig, tsxLoaderConfig]
    }
    return {
        resolve: {
            extensions: ['.ts', '.tsx', 'css', '...'],
        },
        mode: 'production',
        module: {
            rules: [
                ...rules
            ]
        }
    }
}

module.exports = [
    {
        entry: './src/forms/assets_static_src/widgets/index.ts',
        output: {
            'path': path.resolve(__dirname, 'src/forms/assets/static/forms/js/'),
            'filename': 'index.js'
        },
        ...baseConfig([
            {
                test: /\.css$/i,
                use: [
                    'style-loader',
                    'css-loader',
                ]
            },
            {
                test: /\.ts$/i,
                use: [
                    "ts-loader",
                ]
            }
        ]),
    },
    {
        entry: './src/contrib/admin/chooser/static_src/index.ts',
        output: {
            'path': path.resolve(__dirname, 'src/contrib/admin/chooser/assets/static/chooser/js/'),
            'filename': 'index.js'
        },
        ...baseConfig(),
    },
    {
        entry: './src/contrib/images/assets/static_src/index.ts',
        output: {
            'path': path.resolve(__dirname, 'src/contrib/images/assets/static/images/js/'),
            'filename': 'index.js'
        },
        ...baseConfig(),
    },
    {
        entry: './src/contrib/pages/assets/static_src/index.ts',
        output: {
            'path': path.resolve(__dirname, 'src/contrib/pages/assets/static/pages/admin/js/'),
            'filename': 'index.js'
        },
        ...baseConfig(),
    },
    {
        entry: './src/contrib/blocks/assets/static_src/index.ts',
        output: {
            'path': path.resolve(__dirname, 'src/contrib/blocks/assets/static/blocks/js/'),
            'filename': 'index.js'
        },
        ...baseConfig(),
    },
    {
        entry: './src/contrib/editor/static_src/index.ts',
        output: {
            'path': path.resolve(__dirname, 'src/contrib/editor/static/editorjs/js/'),
            'filename': 'index.js'
        },
        ...baseConfig(),
    },
    {
        entry: './src/contrib/editor/features/links/static_src/index.ts',
        output: {
            'path': path.resolve(__dirname, 'src/contrib/editor/features/links/static/links/editorjs/'),
            'filename': 'index.js'
        },
        ...baseConfig([
            tsLoaderConfig,
            tsxLoaderConfig,
            cssLoaderConfig
        ]),
    },
]