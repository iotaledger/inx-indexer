const path = require('path');

module.exports = {
    plugins: [
        [
            '@docusaurus/plugin-content-docs',
            {
                id: 'inx-indexer-develop',
                path: path.resolve(__dirname, 'docs'),
                routeBasePath: 'inx-indexer',
                sidebarPath: path.resolve(__dirname, 'sidebars.js'),
                editUrl: 'https://github.com/iotaledger/inx-indexer/edit/develop/documentation',
                versions: {
                    current: {
                        label: 'Develop',
                        path: 'develop',
                        badge: true
                    },
                },
            }
        ],
    ],
    staticDirectories: [path.resolve(__dirname, 'static')],
};
