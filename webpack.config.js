const path = require('path');

module.exports = {
    entry: './django/assets/static_src/index.ts',
    output: {
        'path': path.resolve(__dirname, 'django/assets/static/django/js/'),
        'filename': 'index.js'
    },
    resolve: {
        extensions: ['.ts', '...'],
    },
    mode: 'production',
    module: {
        rules: [
            {
                test: /\.ts$/i,
                use: 'ts-loader',
                exclude: '/node_modules/'
            }
        ]
    }
}